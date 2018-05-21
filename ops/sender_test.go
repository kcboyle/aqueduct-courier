package ops_test

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/pivotal-cf/aqueduct-courier/ops"
	"github.com/pivotal-cf/aqueduct-courier/ops/opsfakes"
	"github.com/pkg/errors"
)

var _ = Describe("Sender", func() {
	var (
		dataLoader *ghttp.Server
		tarReader  *opsfakes.FakeTarReader
		metadata   Metadata
		tmpFile    *os.File
		tarContent string
		sender     SendExecutor
	)

	BeforeEach(func() {
		dataLoader = ghttp.NewServer()
		sender = SendExecutor{}

		tarReader = new(opsfakes.FakeTarReader)

		metadata = Metadata{
			CollectedAt:  "collected-at",
			CollectionId: "collection-id",
			EnvType:      "some-env-type",
		}
		metadataContents, err := json.Marshal(metadata)
		Expect(err).NotTo(HaveOccurred())

		tmpFile, err = ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
		tarContent = "tar-content"
		_, err = tmpFile.Write([]byte(tarContent))
		Expect(err).NotTo(HaveOccurred())
		Expect(tmpFile.Close()).To(Succeed())

		tarReader.ReadFileStub = func(fileName string) ([]byte, error) {
			if fileName == MetadataFileName {
				return metadataContents, nil
			}

			return []byte{}, errors.New("unexpected file requested")
		}
		tarReader.TarFilePathReturns(tmpFile.Name())

	})

	AfterEach(func() {
		dataLoader.Close()
		Expect(os.RemoveAll(tmpFile.Name())).To(Succeed())
	})

	It("posts to the data loader with the file as content and the file metadata", func() {
		dataLoader.RouteToHandler(http.MethodPost, PostPath, ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				f, fileHeaders, err := req.FormFile("data")
				Expect(err).ToNot(HaveOccurred())
				contents, err := ioutil.ReadAll(f)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal(tarContent))

				metadataStr := req.FormValue("metadata")
				var metadataMap map[string]string
				Expect(json.Unmarshal([]byte(metadataStr), &metadataMap)).To(Succeed())

				Expect(metadataMap["filename"]).To(Equal(fileHeaders.Filename))
				Expect(metadataMap["envType"]).To(Equal(metadata.EnvType))
				Expect(metadataMap["collectedAt"]).To(Equal(metadata.CollectedAt))
				Expect(metadataMap["collectionId"]).To(Equal(metadata.CollectionId))
				Expect(metadataMap["fileContentType"]).To(Equal(TarMimeType))

				md5Sum := md5.Sum([]byte(tarContent))
				Expect(metadataMap["fileMd5Checksum"]).To(Equal(base64.StdEncoding.EncodeToString(md5Sum[:])))
			},
			ghttp.RespondWith(http.StatusCreated, ""),
		))

		Expect(sender.Send(tarReader, dataLoader.URL(), "some-key")).To(Succeed())

		reqs := dataLoader.ReceivedRequests()
		Expect(len(reqs)).To(Equal(1))
	})

	It("posts to the data loader with the correct API key in the header", func() {
		dataLoader.RouteToHandler(http.MethodPost, PostPath, ghttp.CombineHandlers(
			ghttp.VerifyHeader(http.Header{
				"Authorization": []string{"Token some-key"},
			}),
			ghttp.RespondWith(http.StatusCreated, ""),
		))
		Expect(sender.Send(tarReader, dataLoader.URL(), "some-key")).To(Succeed())
	})

	It("errors when the metadata file does not exist", func() {
		tarReader.ReadFileReturns([]byte{}, errors.New("can't find the metadata file"))
		err := sender.Send(tarReader, dataLoader.URL(), "some-key")
		Expect(err).To(MatchError(ContainSubstring(ReadMetadataFileError)))
	})

	It("fails if the metadata file cannot be unmarshalled", func() {
		tarReader.ReadFileReturns([]byte("some-bad-metadata"), nil)

		err := sender.Send(tarReader, dataLoader.URL(), "some-key")
		Expect(err).To(MatchError(ContainSubstring(InvalidMetadataFileError)))
	})

	It("fails if the request object cannot be created", func() {
		err := sender.Send(tarReader, "127.0.0.1:a", "some-key")
		Expect(err).To(MatchError(ContainSubstring(RequestCreationFailureMessage)))
	})

	It("errors when the POST cannot be completed", func() {
		err := sender.Send(tarReader, "http://127.0.0.1:999999", "some-key")
		Expect(err).To(MatchError(ContainSubstring(PostFailedMessage)))
	})

	It("errors when the response code is not StatusCreated", func() {
		dataLoader.AppendHandlers(
			ghttp.RespondWith(http.StatusUnauthorized, ""),
		)

		err := sender.Send(tarReader, dataLoader.URL(), "invalid-key")
		Expect(err).To(MatchError(fmt.Sprintf(UnexpectedResponseCodeFormat, http.StatusUnauthorized)))
	})

	It("when the tarFile does not exist", func() {
		tarReader.TarFilePathReturns("path/to/not/the/tarFile")

		err := sender.Send(tarReader, dataLoader.URL(), "some-key")
		Expect(err).To(MatchError(ContainSubstring(ReadDataFileError)))
	})

	It("fails if the tar file contains more files than what is in the metadata", func() {
		metadata = Metadata{
			FileDigests: []FileDigest{
				{Name: "file1", MD5Checksum: "file1-md5"},
				{Name: "file2", MD5Checksum: "file2-md5"},
			},
		}
		metadataContents, err := json.Marshal(metadata)
		Expect(err).NotTo(HaveOccurred())

		fileMd5s := map[string]string{
			"file1":          "file1-md5",
			"file2":          "file2-md5",
			"too-many-files": "dun dun dunnnnn",
			MetadataFileName: "file-to-skip-checking",
		}

		tarReader.FileMd5sReturns(fileMd5s, nil)
		tarReader.ReadFileReturns(metadataContents, nil)

		err = sender.Send(tarReader, dataLoader.URL(), "token")
		Expect(err).To(MatchError(fmt.Sprintf(ExtraFilesInTarMessageFormat, tarReader.TarFilePath())))
	})

	It("fails if the tar file is missing files listed in the metadata", func() {
		metadata = Metadata{
			FileDigests: []FileDigest{
				{Name: "file1", MD5Checksum: "file1-md5"},
				{Name: "file2", MD5Checksum: "file2-md5"},
			},
		}
		metadataContents, err := json.Marshal(metadata)
		Expect(err).NotTo(HaveOccurred())

		fileMd5s := map[string]string{
			"file1":          "file1-md5",
			MetadataFileName: "file-to-skip-checking",
		}

		tarReader.FileMd5sReturns(fileMd5s, nil)
		tarReader.ReadFileReturns(metadataContents, nil)

		err = sender.Send(tarReader, dataLoader.URL(), "token")
		Expect(err).To(MatchError(fmt.Sprintf(MissingFilesInTarMessageFormat, tarReader.TarFilePath())))
	})

	It("fails if the file checksums in the tarball do not match the metadata", func() {
		metadata = Metadata{
			FileDigests: []FileDigest{
				{Name: "file1", MD5Checksum: "file1-md5"},
				{Name: "file2", MD5Checksum: "file2-md5"},
			},
		}
		metadataContents, err := json.Marshal(metadata)
		Expect(err).NotTo(HaveOccurred())

		fileMd5s := map[string]string{
			"file1":          "file1-md5",
			"file2":          "not-matching-today",
			MetadataFileName: "file-to-skip-checking",
		}

		tarReader.FileMd5sReturns(fileMd5s, nil)
		tarReader.ReadFileReturns(metadataContents, nil)

		err = sender.Send(tarReader, dataLoader.URL(), "token")
		Expect(err).To(MatchError(fmt.Sprintf(InvalidFilesInTarMessageFormat, tarReader.TarFilePath())))
	})

	It("fails if listing the files with their md5s fails", func() {
		tarReader.FileMd5sReturns(map[string]string{}, errors.New("listing files and md5s is hard"))

		err := sender.Send(tarReader, dataLoader.URL(), "token")
		Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf(UnableToListFilesMessageFormat, tarReader.TarFilePath()))))
		Expect(err).To(MatchError(ContainSubstring("listing files and md5s is hard")))
	})
})
