package operations

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/aqueduct-courier/usage"
	uuid "github.com/satori/go.uuid"

	"github.com/pivotal-cf/aqueduct-courier/credhub"

	"github.com/pivotal-cf/aqueduct-courier/opsmanager"
	"github.com/pivotal-cf/aqueduct-utils/data"
	"github.com/pkg/errors"
)

const (
	OpsManagerCollectFailureMessage = "Failed collecting from Operations Manager"
	CredhubCollectFailureMessage    = "Failed collecting from Credhub"
	UsageCollectFailureMessage      = "Failed collecting from Usage Service"
	DataWriteFailureMessage         = "Failed writing data"
	ContentReadingFailureMessage    = "Failed to read content"
)

//go:generate counterfeiter . omDataCollector
type omDataCollector interface {
	Collect() ([]opsmanager.Data, error)
}

//go:generate counterfeiter . credhubDataCollector
type credhubDataCollector interface {
	Collect() (credhub.Data, error)
}

//go:generate counterfeiter . consumptionDataCollector
type consumptionDataCollector interface {
	Collect() (usage.Data, error)
}

//go:generate counterfeiter . tarWriter
type tarWriter interface {
	AddFile([]byte, string) error
	Close() error
}

type CollectExecutor struct {
	opsmanagerDC  omDataCollector
	credhubDC     credhubDataCollector
	consumptionDC consumptionDataCollector
	tarWriter     tarWriter
}

type collectedData interface {
	Name() string
	MimeType() string
	DataType() string
	Type() string
	Content() io.Reader
}

func NewCollector(opsmanagerDC omDataCollector, credhubDC credhubDataCollector, consumptionDC consumptionDataCollector, tarWriter tarWriter) CollectExecutor {
	return CollectExecutor{opsmanagerDC: opsmanagerDC, credhubDC: credhubDC, consumptionDC: consumptionDC, tarWriter: tarWriter}
}

func (ce CollectExecutor) Collect(envType, collectorVersion string) error {
	defer ce.tarWriter.Close()

	opsManagerMetadata := data.Metadata{
		CollectorVersion: collectorVersion,
		EnvType:          envType,
		CollectionId:     uuid.NewV4().String(),
		CollectedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	usageMetadata := data.Metadata{
		CollectorVersion: collectorVersion,
		EnvType:          envType,
		CollectionId:     opsManagerMetadata.CollectionId,
		CollectedAt:      opsManagerMetadata.CollectedAt,
	}

	omDatas, err := ce.opsmanagerDC.Collect()
	if err != nil {
		return errors.Wrap(err, OpsManagerCollectFailureMessage)
	}

	for _, omData := range omDatas {
		err = ce.addData(omData, &opsManagerMetadata, "opsmanager")
		if err != nil {
			return err
		}
	}

	if ce.credhubDC != nil {
		chData, err := ce.credhubDC.Collect()
		if err != nil {
			return errors.Wrap(err, CredhubCollectFailureMessage)
		}

		err = ce.addData(chData, &opsManagerMetadata, "opsmanager")
		if err != nil {
			return err
		}
	}

	metadataContents, err := json.Marshal(opsManagerMetadata)
	if err != nil {
		return err
	}
	err = ce.tarWriter.AddFile(metadataContents, filepath.Join("opsmanager", data.MetadataFileName))
	if err != nil {
		return errors.Wrap(err, DataWriteFailureMessage)
	}

	if ce.consumptionDC != nil {
		usageData, err := ce.consumptionDC.Collect()
		if err != nil {
			return errors.Wrap(err, UsageCollectFailureMessage)
		}

		err = ce.addData(usageData, &usageMetadata, "consumption")
		if err != nil {
			return err
		}

		usageMetadataContents, err := json.Marshal(usageMetadata)
		if err != nil {
			return err
		}

		err = ce.tarWriter.AddFile(usageMetadataContents, filepath.Join("consumption", data.MetadataFileName))
		if err != nil {
			return errors.Wrap(err, DataWriteFailureMessage)
		}
	}

	return nil
}

func (ce CollectExecutor) addData(collectedData collectedData, metadata *data.Metadata, dataSetType string) error {
	dataContents, err := ioutil.ReadAll(collectedData.Content())
	if err != nil {
		return errors.Wrap(err, ContentReadingFailureMessage)
	}

	err = ce.tarWriter.AddFile(dataContents, filepath.Join(dataSetType, collectedData.Name()))
	if err != nil {
		return errors.Wrap(err, DataWriteFailureMessage)
	}

	md5Sum := md5.Sum([]byte(dataContents))
	metadata.FileDigests = append(metadata.FileDigests, data.FileDigest{
		Name:        collectedData.Name(),
		MimeType:    collectedData.MimeType(),
		ProductType: collectedData.Type(),
		DataType:    collectedData.DataType(),
		MD5Checksum: base64.StdEncoding.EncodeToString(md5Sum[:]),
	})
	return nil
}
