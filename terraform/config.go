package terraform

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"os"
)

// CreateConfiguration creates a terraform configuration using the supplied template
func CreateConfiguration(tpl string, objectData any, ownerObjectData any) (string, string, error) {
	td, err := os.MkdirTemp("", "tf-*")
	if err != nil {
		return "", "", err
	}
	f, err := os.CreateTemp(td, "*.tf")
	if err != nil {
		return "", "", err
	}

	t, err := template.New("module").Parse(tpl)
	if err != nil {
		return "", "", err
	}
	err = t.Execute(f, struct{ Object, Owner any }{
		Object: objectData,
		Owner:  ownerObjectData,
	})
	if err != nil {
		return "", "", err
	}

	// create hash of the config
	h := md5.New()
	f.Seek(0, io.SeekStart)
	if _, err := io.Copy(h, f); err != nil {
		return "", "", err
	}
	return td, fmt.Sprintf("%x", h.Sum(nil)), nil
}
