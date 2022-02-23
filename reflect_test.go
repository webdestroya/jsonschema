package jsonschema

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/iancoleman/orderedmap"

	"github.com/invopop/jsonschema/examples"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateFixtures = flag.Bool("update", false, "set to update fixtures")
var compareFixtures = flag.Bool("compare", false, "output failed fixtures with .out.json")

type GrandfatherType struct {
	FamilyName string `json:"family_name" jsonschema:"required"`
}

type SomeBaseType struct {
	SomeBaseProperty     int `json:"some_base_property"`
	SomeBasePropertyYaml int `yaml:"some_base_property_yaml"`
	// The jsonschema required tag is nonsensical for private and ignored properties.
	// Their presence here tests that the fields *will not* be required in the output
	// schema, even if they are tagged required.
	somePrivateBaseProperty   string          `jsonschema:"required"`
	SomeIgnoredBaseProperty   string          `json:"-" jsonschema:"required"`
	SomeSchemaIgnoredProperty string          `jsonschema:"-,required"`
	Grandfather               GrandfatherType `json:"grand"`

	SomeUntaggedBaseProperty           bool `jsonschema:"required"`
	someUnexportedUntaggedBaseProperty bool
}

type MapType map[string]interface{}

type nonExported struct {
	PublicNonExported  int
	privateNonExported int
}

type ProtoEnum int32

func (ProtoEnum) EnumDescriptor() ([]byte, []int) { return []byte(nil), []int{0} }

const (
	Unset ProtoEnum = iota
	Great
)

type TestUser struct {
	SomeBaseType
	nonExported
	MapType

	ID       int                    `json:"id" jsonschema:"required"`
	Name     string                 `json:"name" jsonschema:"required,minLength=1,maxLength=20,pattern=.*,description=this is a property,title=the name,example=joe,example=lucy,default=alex,readOnly=true"`
	Password string                 `json:"password" jsonschema:"writeOnly=true"`
	Friends  []int                  `json:"friends,omitempty" jsonschema_description:"list of IDs, omitted when empty"`
	Tags     map[string]string      `json:"tags,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`

	TestFlag       bool
	IgnoredCounter int `json:"-"`

	// Tests for RFC draft-wright-json-schema-validation-00, section 7.3
	BirthDate time.Time `json:"birth_date,omitempty"`
	Website   url.URL   `json:"website,omitempty"`
	IPAddress net.IP    `json:"network_address,omitempty"`

	// Tests for RFC draft-wright-json-schema-hyperschema-00, section 4
	Photo  []byte `json:"photo,omitempty" jsonschema:"required"`
	Photo2 Bytes  `json:"photo2,omitempty" jsonschema:"required"`

	// Tests for jsonpb enum support
	Feeling ProtoEnum `json:"feeling,omitempty"`
	Age     int       `json:"age" jsonschema:"minimum=18,maximum=120,exclusiveMaximum=true,exclusiveMinimum=true"`
	Email   string    `json:"email" jsonschema:"format=email"`

	// Test for "extras" support
	Baz string `jsonschema_extras:"foo=bar,hello=world,foo=bar1"`

	// Tests for simple enum tags
	Color      string  `json:"color" jsonschema:"enum=red,enum=green,enum=blue"`
	Rank       int     `json:"rank,omitempty" jsonschema:"enum=1,enum=2,enum=3"`
	Multiplier float64 `json:"mult,omitempty" jsonschema:"enum=1.0,enum=1.5,enum=2.0"`

	// Tests for enum tags on slices
	Roles      []string  `json:"roles" jsonschema:"enum=admin,enum=moderator,enum=user"`
	Priorities []int     `json:"priorities,omitempty" jsonschema:"enum=-1,enum=0,enum=1,enun=2"`
	Offsets    []float64 `json:"offsets,omitempty" jsonschema:"enum=1.570796,enum=3.141592,enum=6.283185"`

	// Test for raw JSON
	Anything interface{}     `json:"anything,omitempty"`
	Raw      json.RawMessage `json:"raw"`
}

type CustomTime time.Time

type CustomTypeField struct {
	CreatedAt CustomTime
}

type CustomTimeWithInterface time.Time

type CustomTypeFieldWithInterface struct {
	CreatedAt CustomTimeWithInterface
}

func (CustomTimeWithInterface) JSONSchema() *Schema {
	return &Schema{
		Type:   "string",
		Format: "date-time",
	}
}

type RootOneOf struct {
	Field1 string      `json:"field1" jsonschema:"oneof_required=group1"`
	Field2 string      `json:"field2" jsonschema:"oneof_required=group2"`
	Field3 interface{} `json:"field3" jsonschema:"oneof_type=string;array"`
	Field4 string      `json:"field4" jsonschema:"oneof_required=group1"`
	Field5 ChildOneOf  `json:"child"`
}

type ChildOneOf struct {
	Child1 string      `json:"child1" jsonschema:"oneof_required=group1"`
	Child2 string      `json:"child2" jsonschema:"oneof_required=group2"`
	Child3 interface{} `json:"child3" jsonschema:"oneof_required=group2,oneof_type=string;array"`
	Child4 string      `json:"child4" jsonschema:"oneof_required=group1"`
}

type Outer struct {
	Inner
}

type Inner struct {
	Foo string `yaml:"foo"`
}

type MinValue struct {
	Value int `json:"value4" jsonschema_extras:"minimum=0"`
}
type Bytes []byte

type TestNullable struct {
	Child1 string `json:"child1" jsonschema:"nullable"`
}

type TestYamlInline struct {
	Inlined Inner `yaml:",inline"`
}

type TestYamlAndJson struct {
	FirstName  string `json:"FirstName" yaml:"first_name"`
	LastName   string `json:"LastName"`
	Age        uint   `yaml:"age"`
	MiddleName string `yaml:"middle_name,omitempty" json:"MiddleName,omitempty"`
}

type CompactDate struct {
	Year  int
	Month int
}

type UserWithAnchor struct {
	Name string `json:"name" jsonschema:"anchor=Name"`
}

func (CompactDate) JSONSchema() *Schema {
	return &Schema{
		Type:        "string",
		Title:       "Compact Date",
		Description: "Short date that only includes year and month",
		Pattern:     "^[0-9]{4}-[0-1][0-9]$",
	}
}

type TestYamlAndJson2 struct {
	FirstName  string `json:"FirstName" yaml:"first_name"`
	LastName   string `json:"LastName"`
	Age        uint   `yaml:"age"`
	MiddleName string `yaml:"middle_name,omitempty" json:"MiddleName,omitempty"`
}

func (TestYamlAndJson2) GetFieldDocString(fieldName string) string {
	switch fieldName {
	case "FirstName":
		return "test2"
	case "LastName":
		return "test3"
	case "Age":
		return "test4"
	case "MiddleName":
		return "test5"
	default:
		return ""
	}
}

type LookupName struct {
	Given   string `json:"first"`
	Surname string `json:"surname"`
}

type LookupUser struct {
	Name  *LookupName `json:"name"`
	Alias string      `json:"alias,omitempty"`
}

type CustomSliceOuter struct {
	Slice CustomSliceType `json:"slice"`
}

type CustomSliceType []string

func (CustomSliceType) JSONSchema() *Schema {
	return &Schema{
		OneOf: []*Schema{{
			Type: "string",
		}, {
			Type: "array",
			Items: &Schema{
				Type: "string",
			},
		}},
	}
}

type CustomMapType map[string]string

func (CustomMapType) JSONSchema() *Schema {
	properties := orderedmap.New()
	properties.Set("key", &Schema{
		Type: "string",
	})
	properties.Set("value", &Schema{
		Type: "string",
	})
	return &Schema{
		Type: "array",
		Items: &Schema{
			Type:       "object",
			Properties: properties,
			Required:   []string{"key", "value"},
		},
	}
}

type CustomMapOuter struct {
	MyMap CustomMapType `json:"my_map"`
}

type PatternTest struct {
	WithPattern string `json:"with_pattern" jsonschema:"minLength=1,pattern=[0-9]{1\\,4},maxLength=50"`
}

func TestReflector(t *testing.T) {
	r := new(Reflector)
	s := "http://example.com/schema"
	r.SetBaseSchemaID(s)
	assert.EqualValues(t, s, r.BaseSchemaID)
}

func TestReflectFromType(t *testing.T) {
	r := new(Reflector)
	tu := new(TestUser)
	typ := reflect.TypeOf(tu)

	s := r.ReflectFromType(typ)
	assert.EqualValues(t, "https://github.com/invopop/jsonschema/test-user", s.ID)

	x := struct {
		Test string
	}{
		Test: "foo",
	}
	typ = reflect.TypeOf(x)
	s = r.Reflect(typ)
	assert.Empty(t, s.ID)
}

func TestSchemaGeneration(t *testing.T) {
	tests := []struct {
		typ       interface{}
		reflector *Reflector
		fixture   string
	}{
		{&TestUser{}, &Reflector{}, "fixtures/test_user.json"},
		{&UserWithAnchor{}, &Reflector{}, "fixtures/user_with_anchor.json"},
		{&TestUser{}, &Reflector{AssignAnchor: true}, "fixtures/test_user_assign_anchor.json"},
		{&TestUser{}, &Reflector{AllowAdditionalProperties: true}, "fixtures/allow_additional_props.json"},
		{&TestUser{}, &Reflector{RequiredFromJSONSchemaTags: true}, "fixtures/required_from_jsontags.json"},
		{&TestUser{}, &Reflector{ExpandedStruct: true}, "fixtures/defaults_expanded_toplevel.json"},
		{&TestUser{}, &Reflector{IgnoredTypes: []interface{}{GrandfatherType{}}}, "fixtures/ignore_type.json"},
		{&TestUser{}, &Reflector{DoNotReference: true}, "fixtures/no_reference.json"},
		{&TestUser{}, &Reflector{DoNotReference: true, AssignAnchor: true}, "fixtures/no_reference_anchor.json"},
		{&RootOneOf{}, &Reflector{RequiredFromJSONSchemaTags: true}, "fixtures/oneof.json"},
		{&CustomTypeField{}, &Reflector{
			Mapper: func(i reflect.Type) *Schema {
				if i == reflect.TypeOf(CustomTime{}) {
					return &Schema{
						Type:   "string",
						Format: "date-time",
					}
				}
				return nil
			},
		}, "fixtures/custom_type.json"},
		{LookupUser{}, &Reflector{BaseSchemaID: "https://example.com/schemas"}, "fixtures/base_schema_id.json"},
		{LookupUser{}, &Reflector{
			BaseSchemaID: "https://example.com/schemas",
			Lookup: func(i reflect.Type) ID {
				switch i {
				case reflect.TypeOf(LookupUser{}):
					return ID("https://example.com/schemas/lookup-user")
				case reflect.TypeOf(LookupName{}):
					return ID("https://example.com/schemas/lookup-name")
				}
				return EmptyID
			},
		}, "fixtures/lookup.json"},
		{&LookupUser{}, &Reflector{
			BaseSchemaID:   "https://example.com/schemas",
			ExpandedStruct: true,
			AssignAnchor:   true,
			Lookup: func(i reflect.Type) ID {
				switch i {
				case reflect.TypeOf(LookupUser{}):
					return ID("https://example.com/schemas/lookup-user")
				case reflect.TypeOf(LookupName{}):
					return ID("https://example.com/schemas/lookup-name")
				}
				return EmptyID
			},
		}, "fixtures/lookup_expanded.json"},
		{&Outer{}, &Reflector{ExpandedStruct: true, DoNotReference: true, YAMLEmbeddedStructs: true}, "fixtures/disable_inlining_embedded.json"},
		{&Outer{}, &Reflector{ExpandedStruct: true, DoNotReference: true, YAMLEmbeddedStructs: true, AssignAnchor: true}, "fixtures/disable_inlining_embedded_anchored.json"},
		{&MinValue{}, &Reflector{}, "fixtures/schema_with_minimum.json"},
		{&TestNullable{}, &Reflector{}, "fixtures/nullable.json"},
		{&TestYamlInline{}, &Reflector{YAMLEmbeddedStructs: true}, "fixtures/yaml_inline_embed.json"},
		{&TestYamlInline{}, &Reflector{}, "fixtures/yaml_inline_embed.json"},
		{&GrandfatherType{}, &Reflector{
			AdditionalFields: func(r reflect.Type) []reflect.StructField {
				return []reflect.StructField{
					{
						Name:      "Addr",
						Type:      reflect.TypeOf((*net.IP)(nil)).Elem(),
						Tag:       "json:\"ip_addr\"",
						Anonymous: false,
					},
				}
			},
		}, "fixtures/custom_additional.json"},
		{&TestYamlAndJson{}, &Reflector{PreferYAMLSchema: true}, "fixtures/test_yaml_and_json_prefer_yaml.json"},
		{&TestYamlAndJson{}, &Reflector{}, "fixtures/test_yaml_and_json.json"},
		// {&TestYamlAndJson2{}, &Reflector{}, "fixtures/test_yaml_and_json2.json"},
		{&CompactDate{}, &Reflector{}, "fixtures/compact_date.json"},
		{&CustomSliceOuter{}, &Reflector{}, "fixtures/custom_slice_type.json"},
		{&CustomMapOuter{}, &Reflector{}, "fixtures/custom_map_type.json"},
		{&CustomTypeFieldWithInterface{}, &Reflector{}, "fixtures/custom_type_with_interface.json"},
		{&PatternTest{}, &Reflector{}, "fixtures/commas_in_pattern.json"},
		{&examples.User{}, prepareCommentReflector(t), "fixtures/go_comments.json"},
	}

	for _, tt := range tests {
		name := strings.TrimSuffix(filepath.Base(tt.fixture), ".json")
		t.Run(name, func(t *testing.T) {
			compareSchemaOutput(t,
				tt.fixture, tt.reflector, tt.typ,
			)
		})
	}
}

func prepareCommentReflector(t *testing.T) *Reflector {
	t.Helper()
	r := new(Reflector)
	err := r.AddGoComments("github.com/invopop/jsonschema", "./examples")
	require.NoError(t, err, "did not expect error while adding comments")
	return r
}

func TestBaselineUnmarshal(t *testing.T) {
	r := &Reflector{}
	compareSchemaOutput(t, "fixtures/test_user.json", r, &TestUser{})
}

func compareSchemaOutput(t *testing.T, f string, r *Reflector, obj interface{}) {
	t.Helper()
	expectedJSON, err := ioutil.ReadFile(f)
	require.NoError(t, err)

	actualSchema := r.Reflect(obj)
	actualJSON, _ := json.MarshalIndent(actualSchema, "", "  ")

	if *updateFixtures {
		_ = ioutil.WriteFile(f, actualJSON, 0600)
	}

	if !assert.JSONEq(t, string(expectedJSON), string(actualJSON)) {
		if *compareFixtures {
			_ = ioutil.WriteFile(strings.TrimSuffix(f, ".json")+".out.json", actualJSON, 0600)
		}
	}
}

func TestSplitOnUnescapedCommas(t *testing.T) {
	tests := []struct {
		strToSplit string
		expected   []string
	}{
		{`Hello,this,is\,a\,string,haha`, []string{`Hello`, `this`, `is,a,string`, `haha`}},
		{`hello,no\\,split`, []string{`hello`, `no\,split`}},
		{`string without commas`, []string{`string without commas`}},
		{`ünicode,𐂄,Ж\,П,ᠳ`, []string{`ünicode`, `𐂄`, `Ж,П`, `ᠳ`}},
		{`empty,,tag`, []string{`empty`, ``, `tag`}},
	}

	for _, test := range tests {
		actual := splitOnUnescapedCommas(test.strToSplit)
		require.Equal(t, test.expected, actual)
	}
}
