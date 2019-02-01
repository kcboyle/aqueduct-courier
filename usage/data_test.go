package usage_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/aqueduct-courier/usage"
	"github.com/pivotal-cf/aqueduct-utils/data"
)

var _ = Describe("Data", func() {

	It("returns a the data type for the name", func() {
		d := NewData(strings.NewReader(""), data.AppUsagesDataType)
		Expect(d.Name()).To(Equal(data.AppUsagesDataType))
	})

	It("returns content for the data", func() {
		dataReader := strings.NewReader("best-data")
		d := NewData(dataReader, data.AppUsagesDataType)
		Expect(d.Content()).To(Equal(dataReader))
	})

	It("returns json as data type", func() {
		d := NewData(nil, data.AppUsagesDataType)
		Expect(d.MimeType()).To(Equal("application/json"))
	})

	It("returns the product type", func() {
		d := NewData(nil, data.AppUsagesDataType)
		Expect(d.Type()).To(Equal(""))
	})

	It("returns the data type", func() {
		d := NewData(nil, data.AppUsagesDataType)
		Expect(d.DataType()).To(Equal(data.AppUsagesDataType))
	})

})
