package pubsub

import (
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/redhat-cne/sdk-go/pkg/common"
	log "github.com/sirupsen/logrus"
)

var iterPool = sync.Pool{
	New: func() interface{} {
		return jsoniter.Parse(jsoniter.ConfigFastest, nil, 1024)
	},
}

func borrowIterator(reader io.Reader) *jsoniter.Iterator {
	iter := iterPool.Get().(*jsoniter.Iterator)
	iter.Reset(reader)
	return iter
}

func returnIterator(iter *jsoniter.Iterator) {
	iter.Error = nil
	iter.Attachment = nil
	iterPool.Put(iter)
}

// ReadJSON ...
func ReadJSON(out *PubSub, reader io.Reader) error {
	iterator := borrowIterator(reader)
	defer returnIterator(iterator)
	return readJSONFromIterator(out, iterator)
}

// readJSONFromIterator allows you to read the bytes reader as an PubSub
func readJSONFromIterator(out *PubSub, iterator *jsoniter.Iterator) error {
	var (
		id          string
		endpointUri string //nolint:revive
		uriLocation string
		resource    string
		version     string
	)

	for key := iterator.ReadObject(); key != ""; key = iterator.ReadObject() {
		// Check if we have some error in our error cache
		if iterator.Error != nil {
			return iterator.Error
		}

		// If no specversion ...
		switch key {
		case "SubscriptionId":
			id = iterator.ReadString()
		case "EndpointUri":
			endpointUri = iterator.ReadString()
		case "UriLocation":
			uriLocation = iterator.ReadString()
		case "ResourceAddress":
			resource = iterator.ReadString()
			version = common.V2
		case "id":
			id = iterator.ReadString()
		case "endpointUri":
			endpointUri = iterator.ReadString()
		case "uriLocation":
			uriLocation = iterator.ReadString()
		case "resource":
			log.Warningf("%s, resource is used instead of ResourceAddress", common.NONCOMPLIANT)
			resource = iterator.ReadString()
			version = common.V1
		default:
			iterator.Skip()
		}
	}

	if iterator.Error != nil {
		return iterator.Error
	}

	out.SetID(id)
	out.SetEndpointURI(endpointUri) //nolint:errcheck
	out.SetURILocation(uriLocation) //nolint:errcheck
	out.SetResource(resource)       //nolint:errcheck
	out.SetVersion(version)

	return nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (d *PubSub) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readJSONFromIterator(d, iterator)
}
