package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
)

// StringSlice wraps a []string to satisfy flag.Value
type StringSlice struct {
	slice      []string
	hasBeenSet bool
}

// NewStringSlice creates a *StringSlice with default values
func NewStringSlice(defaults ...string) *StringSlice {
	return &StringSlice{slice: append([]string{}, defaults...)}
}

// clone allocate a copy of self object
func (s *StringSlice) clone() *StringSlice {
	n := &StringSlice{
		slice:      make([]string, len(s.slice)),
		hasBeenSet: s.hasBeenSet,
	}
	copy(n.slice, s.slice)
	return n
}

// Set appends the string value to the list of values
func (s *StringSlice) Set(value string) error {
	if !s.hasBeenSet {
		s.slice = []string{}
		s.hasBeenSet = true
	}

	for _, v := range strings.Split(value, ",") {
		s.slice = append(s.slice, strings.TrimSpace(v))
	}
	return nil
}

// String returns a readable representation of this value (for usage defaults)
func (s *StringSlice) String() string {
	return fmt.Sprintf("%s", s.slice)
}

// Serialize allows StringSlice to fulfill Serializer
func (s *StringSlice) Serialize() string {
	jsonBytes, _ := json.Marshal(s.slice)
	return fmt.Sprintf("%s%s", slPfx, string(jsonBytes))
}

// Value returns the slice of strings set by this flag
func (s *StringSlice) Value() []string {
	return s.slice
}

// Get returns the slice of strings set by this flag
func (s *StringSlice) Get() interface{} {
	return *s
}

// StringSliceFlag is a flag with type *StringSlice
type StringSliceFlag struct {
	Name        string
	Aliases     []string
	Usage       string
	EnvVars     []string
	FilePath    string
	Required    bool
	Hidden      bool
	TakesFile   bool
	Value       *StringSlice
	DefaultText string
	HasBeenSet  bool
	Destination *StringSlice
}

// IsSet returns whether or not the flag has been set through env or file
func (f *StringSliceFlag) IsSet() bool {
	return f.HasBeenSet
}

// String returns a readable representation of this value
// (for usage defaults)
func (f *StringSliceFlag) String() string {
	return FlagStringer(f)
}

// Names returns the names of the flag
func (f *StringSliceFlag) Names() []string {
	return flagNames(f.Name, f.Aliases)
}

// IsRequired returns whether or not the flag is required
func (f *StringSliceFlag) IsRequired() bool {
	return f.Required
}

// TakesValue returns true of the flag takes a value, otherwise false
func (f *StringSliceFlag) TakesValue() bool {
	return true
}

// GetUsage returns the usage string for the flag
func (f *StringSliceFlag) GetUsage() string {
	return f.Usage
}

// GetValue returns the flags value as string representation and an empty
// string if the flag takes no value at all.
func (f *StringSliceFlag) GetValue() string {
	if f.Value != nil {
		return f.Value.String()
	}
	return ""
}

// IsVisible returns true if the flag is not hidden, otherwise false
func (f *StringSliceFlag) IsVisible() bool {
	return !f.Hidden
}

// Apply populates the flag given the flag set and environment
func (f *StringSliceFlag) Apply(set *flag.FlagSet) error {

	if f.Destination != nil && f.Value != nil {
		f.Destination.slice = make([]string, len(f.Value.slice))
		copy(f.Destination.slice, f.Value.slice)

	}

	if val, ok := flagFromEnvOrFile(f.EnvVars, f.FilePath); ok {
		if f.Value == nil {
			f.Value = &StringSlice{}
		}
		destination := f.Value
		if f.Destination != nil {
			destination = f.Destination
		}

		for _, s := range strings.Split(val, ",") {
			if err := destination.Set(strings.TrimSpace(s)); err != nil {
				return fmt.Errorf("could not parse %q as string value for flag %s: %s", val, f.Name, err)
			}
		}

		// Set this to false so that we reset the slice if we then set values from
		// flags that have already been set by the environment.
		destination.hasBeenSet = false
		f.HasBeenSet = true
	}

	if f.Value == nil {
		f.Value = &StringSlice{}
	}
	setValue := f.Destination
	if f.Destination == nil {
		setValue = f.Value.clone()
	}
	for _, name := range f.Names() {
		set.Var(setValue, name, f.Usage)
	}

	return nil
}

// StringSlice looks up the value of a local StringSliceFlag, returns
// nil if not found
func (c *Context) StringSlice(name string) []string {
	if fs := c.lookupFlagSet(name); fs != nil {
		return lookupStringSlice(name, fs)
	}
	return nil
}

// 列表解码为 map, -flag="K1=V1,K2=V2"
func (c *Context) StringSliceAsMap(name string) map[string]string {
	var m = make(map[string]string)
	for _, kv := range c.StringSlice(name) {
		if idx := strings.Index(kv, "="); idx >= 0 {
			k, v := kv[:idx], kv[idx+1:]
			m[k] = v
		} else {
			m[kv] = ""
		}
	}
	return m
}

func lookupStringSlice(name string, set *flag.FlagSet) []string {
	f := set.Lookup(name)
	if f != nil {
		if slice, ok := f.Value.(*StringSlice); ok {
			return slice.Value()
		}
	}
	return nil
}
