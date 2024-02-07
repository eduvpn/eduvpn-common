// package ini implements an opinionated ini parser that only implements what we exactly need for WireGuard configs
// - key/values MUST live under a section
// - empty section names are NOT allowed
// - comments are indicated with a #
package ini

import (
	"errors"
	"fmt"
	"strings"
)

// shouldSkip returns whether or not a line should be skipped, empty line (after whitespace omitting) or a comment
func shouldSkip(f string) bool {
	return f == "" || strings.HasPrefix(f, "#")
}

// isSection returns whether a line is a section
// this happens when it begins with [ and ends with ]
func isSection(f string) bool {
	return strings.HasPrefix(f, "[") && strings.HasSuffix(f, "]")
}

// sectionName extracts the section name from a line by removing the [ and ] prefix and suffix
func sectionName(f string) string {
	name := strings.TrimSuffix(strings.TrimPrefix(f, "["), "]")
	return strings.TrimSpace(name)
}

// keyValue extracts a key and a value from a line
// if no 2 components are found (separated by =), we will return an error
// the key and value have their spaces trimmed
func keyValue(f string) (string, string, error) {
	sl := strings.SplitN(f, "=", 2)
	if len(sl) < 2 {
		return "", "", errors.New("no key/value found")
	}
	k := strings.TrimSpace(sl[0])
	if k == "" {
		return "", "", errors.New("key cannot be empty")
	}
	v := strings.TrimSpace(sl[1])
	return k, v, nil
}

// OrderedKeys is a slice of strings that is used for an ordered map
type OrderedKeys []string

func (ok *OrderedKeys) find(name string) int {
	if ok == nil {
		return -1
	}
	for i, v := range *ok {
		if v == name {
			return i
		}
	}
	return -1
}

// Remove removes a `name` from the OrderedKeys slice by finding the name
// It is a no-op if the key does not exist
func (ok *OrderedKeys) Remove(name string) {
	idx := ok.find(name)
	if idx == -1 {
		return
	}
	*ok = append((*ok)[:idx], (*ok)[idx+1:]...)
}

// Section represents a single section within an ini file
// It consists of multiple key and values
type Section struct {
	keyValues map[string]string
	keys      OrderedKeys
}

// KeyValue gets a value for key `key`
// It returns an error if the key does not exist
func (sec *Section) KeyValue(key string) (string, error) {
	if v, ok := sec.keyValues[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key: '%s' does not exist", key)
}

func (sec *Section) newKeyValue(key string, value string) {
	if sec.keyValues == nil {
		sec.keyValues = make(map[string]string)
	}
	sec.keyValues[key] = value
	sec.keys = append(sec.keys, key)
}

// AddOrReplaceKeyValue adds a key `key` with value `value`
// If the key already exists it modifies the value
func (sec *Section) AddOrReplaceKeyValue(key string, value string) {
	_, err := sec.KeyValue(key)
	if err == nil {
		sec.keyValues[key] = value
		return
	}
	sec.newKeyValue(key, value)
}

// AddKeyValue adds a new key `key` with value `value`
// It returns an error if the key already exists
func (sec *Section) AddKeyValue(key string, value string) error {
	// get an existing key
	_, err := sec.KeyValue(key)
	if err == nil {
		return fmt.Errorf("key: '%s' already exists", key)
	}
	sec.newKeyValue(key, value)
	return nil
}

// RemoveKey removes a key `key` from the section
// It returns an error if the key cannot be found
func (sec *Section) RemoveKey(key string) (string, error) {
	if v, ok := sec.keyValues[key]; ok {
		sec.keys.Remove(key)
		delete(sec.keyValues, key)
		return v, nil
	}
	return "", fmt.Errorf("no key to remove with name: '%s'", key)
}

// INI is the struct for a ini file
type INI struct {
	sections map[string]*Section
	keys     OrderedKeys
}

// Empty returns true if there are no sections defined in the INI
func (i *INI) Empty() bool {
	return len(i.keys) == 0
}

// Section gets a section from the ini file
func (i *INI) Section(name string) (*Section, error) {
	if _, ok := i.sections[name]; ok {
		return i.sections[name], nil
	}
	return nil, fmt.Errorf("section: '%s' does not exist", name)
}

// AddSection adds a section with name `name` and returns an error if the section already exists
func (i *INI) AddSection(name string) error {
	// get an existing section
	_, err := i.Section(name)
	if err == nil {
		return errors.New("section: '%s' already exists")
	}
	if i.sections == nil {
		i.sections = make(map[string]*Section)
	}
	i.sections[name] = &Section{}
	i.keys = append(i.keys, name)
	return nil
}

// String returns the representation of the ini as a string
func (i *INI) String() string {
	var out strings.Builder
	for _, s := range i.keys {
		sec, err := i.Section(s)
		if err != nil {
			continue
		}
		out.WriteString(fmt.Sprintf("[%s]\n", s))

		for _, k := range sec.keys {
			v, err := sec.KeyValue(k)
			if err != nil {
				continue
			}
			delim := ""
			if v != "" {
				delim = " "
			}
			out.WriteString(fmt.Sprintf("%s =%s%s\n", k, delim, v))
		}
	}
	return out.String()
}

// Parse returns a slice of sections
// we do not return a map as we want to ensure the same ordering of sections, keys and values
func Parse(f string) INI {
	lines := strings.Split(f, "\n")

	var secs INI
	sec := ""
	for _, line := range lines {
		// clean the line
		line = strings.TrimSpace(line)

		if shouldSkip(line) {
			continue
		}

		if isSection(line) {
			name := sectionName(line)
			// we do not allow sections with empty names
			if name == "" {
				continue
			}
			_ = secs.AddSection(name)
			sec = name
			continue
		}

		// no section has been parsed yet
		// we will ignore the rest of the values
		if sec == "" {
			continue
		}

		// split key and value
		key, value, err := keyValue(line)
		if err != nil {
			continue
		}

		csec := secs.sections[sec]
		// This adds a new key and value to the section
		// If it already exists it ignores it as this function would return an error
		_ = csec.AddKeyValue(key, value)
	}
	return secs
}
