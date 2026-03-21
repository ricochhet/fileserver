package xmlutil

import (
	"encoding/xml"
	"os"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
)

// ReadAndUnmarshal parses an XML file from the specified path.
func ReadAndUnmarshal[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errutil.New("os.ReadFile", err)
	}

	var t T
	if err := xml.Unmarshal(data, &t); err != nil {
		return nil, errutil.New("xml.Unmarshal", err)
	}

	return &t, nil
}

// MarshalAndWrite marshales the data to the specified output file.
func MarshalAndWrite[T any](path string, data T) ([]byte, error) {
	b, err := xml.MarshalIndent(data, "", "\t")
	if err != nil {
		return nil, err
	}

	return b, fsutil.Write(path, b)
}
