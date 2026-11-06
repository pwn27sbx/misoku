package config

import "encoding/json"

// SetLearnOffered records in the config file at path that the first-run
// dialog was answered, so it never shows again on this machine. It reports
// no error when the key is already true, and leaves the file byte for byte
// then; every other setting stays as written, those this version of Misoku
// does not know included (see editFile).
func SetLearnOffered(path string) error {
	_, err := editFile(path, func(top map[string]json.RawMessage) (bool, error) {
		if raw, ok := top["learn_offered"]; ok && string(raw) == "true" {
			return false, nil
		}
		top["learn_offered"] = json.RawMessage("true")
		return true, nil
	})
	return err
}
