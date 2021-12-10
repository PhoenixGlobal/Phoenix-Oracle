package models

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm/schema"

	"gorm.io/gorm"

	"PhoenixOracle/core/assets"
	"github.com/araddon/dateparse"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var CronParser cron.Parser

func init() {
	cronParserSpec := cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor
	CronParser = cron.NewParser(cronParserSpec)
}

type JSON struct {
	gjson.Result
}

func (JSON) GormDataType() string {
	return "json"
}

func (JSON) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "JSONB"
	}
	return ""
}

func (j JSON) Value() (driver.Value, error) {
	s := j.Bytes()
	if len(s) == 0 {
		return nil, nil
	}
	return s, nil
}

func (j *JSON) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*j = JSON{Result: gjson.Parse(v)}
	case []byte:
		*j = JSON{Result: gjson.ParseBytes(v)}
	default:
		return fmt.Errorf("unable to convert %v of %T to JSON", value, value)
	}
	return nil
}

func MustParseJSON(b []byte) JSON {
	var j JSON
	str := string(b)
	if len(str) == 0 {
		panic("empty byte array")
	}
	if err := json.Unmarshal([]byte(str), &j); err != nil {
		panic(err)
	}
	return j
}

func ParseJSON(b []byte) (JSON, error) {
	var j JSON
	str := string(b)
	if len(str) == 0 {
		return j, nil
	}
	return j, json.Unmarshal([]byte(str), &j)
}

func (j *JSON) UnmarshalJSON(b []byte) error {
	str := string(b)
	if !gjson.Valid(str) {
		return fmt.Errorf("invalid JSON: %v", str)
	}
	*j = JSON{gjson.Parse(str)}
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if j.Exists() {
		return j.Bytes(), nil
	}
	return []byte("{}"), nil
}

func (j *JSON) UnmarshalTOML(val interface{}) error {
	var bs []byte
	switch v := val.(type) {
	case string:
		bs = []byte(v)
	case []byte:
		bs = v
	}
	var err error
	*j, err = ParseJSON(bs)
	return err
}

func (j JSON) Bytes() []byte {
	if len(j.String()) == 0 {
		return nil
	}
	return []byte(j.String())
}

func (j JSON) AsMap() (map[string]interface{}, error) {
	output := make(map[string]interface{})
	switch v := j.Result.Value().(type) {
	case map[string]interface{}:
		for key, value := range v {
			output[key] = value
		}
	case nil:
	default:
		return nil, errors.New("can only add to JSON objects or null")
	}
	return output, nil
}

func mapToJSON(m map[string]interface{}) (JSON, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return JSON{}, err
	}
	return JSON{Result: gjson.ParseBytes(bytes)}, nil
}

func (j JSON) Add(insertKey string, insertValue interface{}) (JSON, error) {
	return j.MultiAdd(KV{insertKey: insertValue})
}

func (j JSON) PrependAtArrayKey(insertKey string, insertValue interface{}) (JSON, error) {
	curr := j.Get(insertKey).Array()
	updated := make([]interface{}, 0)
	updated = append(updated, insertValue)
	for _, c := range curr {
		updated = append(updated, c.Value())
	}
	return j.Add(insertKey, updated)
}

type KV map[string]interface{}

func (j JSON) MultiAdd(keyValues KV) (JSON, error) {
	output, err := j.AsMap()
	if err != nil {
		return JSON{}, err
	}
	for key, value := range keyValues {
		output[key] = value
	}
	return mapToJSON(output)
}

func (j JSON) Delete(key string) (JSON, error) {
	js, err := sjson.Delete(j.String(), key)
	if err != nil {
		return j, err
	}
	return ParseJSON([]byte(js))
}

func (j JSON) CBOR() ([]byte, error) {
	switch v := j.Result.Value().(type) {
	case map[string]interface{}, []interface{}, nil:
		return cbor.Marshal(v)
	default:
		var b []byte
		return b, fmt.Errorf("unable to coerce JSON to CBOR for type %T", v)
	}
}

func MarshalToMap(input interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var output map[string]interface{}
	err = json.Unmarshal(bytes, &output)
	if err != nil {
		// Technically this should be impossible
		return nil, err
	}
	return output, nil
}

// WebURL contains the URL of the endpoint.
type WebURL url.URL

func (w *WebURL) UnmarshalJSON(j []byte) error {
	var v string
	err := json.Unmarshal(j, &v)
	if err != nil {
		return err
	}
	// handle no url case
	if len(v) == 0 {
		return nil
	}

	u, err := url.ParseRequestURI(v)
	if err != nil {
		return err
	}
	*w = WebURL(*u)
	return nil
}

func (w WebURL) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.String())
}

func (w WebURL) String() string {
	url := url.URL(w)
	return url.String()
}

func (w WebURL) Value() (driver.Value, error) {
	return w.String(), nil
}

func (w *WebURL) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to WebURL", value, value)
	}

	u, err := url.ParseRequestURI(s)
	if err != nil {
		return err
	}
	*w = WebURL(*u)
	return nil
}

type AnyTime struct {
	time.Time
	Valid bool
}

func NewAnyTime(t time.Time) AnyTime {
	return AnyTime{Time: t, Valid: true}
}

func (t AnyTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return t.Time.UTC().MarshalJSON()
}

func (t *AnyTime) UnmarshalJSON(b []byte) error {
	var str string

	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		str = n.String()
	} else if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	if len(str) == 0 {
		t.Valid = false
		return nil
	}

	newTime, err := dateparse.ParseAny(str)
	t.Time = newTime.UTC()
	t.Valid = true
	return err
}

func (t AnyTime) MarshalText() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return t.Time.MarshalText()
}

func (t *AnyTime) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		t.Valid = false
		return nil
	}
	if err := t.Time.UnmarshalText(text); err != nil {
		return err
	}
	t.Valid = true
	return nil
}

func (t AnyTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}
	return t.Time, nil
}

func (t *AnyTime) Scan(value interface{}) error {
	switch temp := value.(type) {
	case time.Time:
		t.Time = temp.UTC()
		t.Valid = true
		return nil
	case nil:
		t.Valid = false
		return nil
	default:
		return fmt.Errorf("unable to convert %v of %T to Time", value, value)
	}
}

type Cron string

func (c *Cron) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return fmt.Errorf("Cron: %w", err)
	}
	if s == "" {
		return nil
	}

	if !strings.HasPrefix(s, "CRON_TZ=") {
		return errors.New("Cron: specs must specify a time zone using CRON_TZ, e.g. 'CRON_TZ=UTC 5 * * * *'")
	}

	_, err = CronParser.Parse(s)
	if err != nil {
		return fmt.Errorf("Cron: %w", err)
	}
	*c = Cron(s)
	return nil
}

func (c Cron) String() string {
	return string(c)
}

type Duration struct{ d time.Duration }

func MakeDuration(d time.Duration) (Duration, error) {
	if d < time.Duration(0) {
		return Duration{}, fmt.Errorf("cannot make negative time duration: %s", d)
	}
	return Duration{d: d}, nil
}

func MustMakeDuration(d time.Duration) Duration {
	rv, err := MakeDuration(d)
	if err != nil {
		panic(err)
	}
	return rv
}

func (d Duration) Duration() time.Duration {
	return d.d
}

func (d Duration) Before(t time.Time) time.Time {
	return t.Add(-d.Duration())
}

func (d Duration) Shorter(od Duration) bool { return d.d < od.d }

func (d Duration) IsInstant() bool { return d.d == 0 }

func (d Duration) String() string {
	return d.Duration().String()
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(input []byte) error {
	var txt string
	err := json.Unmarshal(input, &txt)
	if err != nil {
		return err
	}
	v, err := time.ParseDuration(string(txt))
	if err != nil {
		return err
	}
	*d, err = MakeDuration(v)
	if err != nil {
		return err
	}
	return nil
}

func (d *Duration) Scan(v interface{}) (err error) {
	switch tv := v.(type) {
	case int64:
		*d, err = MakeDuration(time.Duration(tv))
		return err
	default:
		return errors.Errorf(`don't know how to parse "%s" of type %T as a `+
			`models.Duration`, tv, tv)
	}
}

func (d Duration) Value() (driver.Value, error) {
	return int64(d.d), nil
}

type Interval time.Duration

func (i Interval) Duration() time.Duration {
	return time.Duration(i)
}

func (i Interval) MarshalText() ([]byte, error) {
	return []byte(time.Duration(i).String()), nil
}

func (i *Interval) UnmarshalText(input []byte) error {
	v, err := time.ParseDuration(string(input))
	if err != nil {
		return err
	}
	*i = Interval(v)
	return nil
}

func (i *Interval) Scan(v interface{}) error {
	if v == nil {
		*i = Interval(time.Duration(0))
		return nil
	}
	asInt64, is := v.(int64)
	if !is {
		return errors.Errorf("models.Interval#Scan() wanted int64, got %T", v)
	}
	*i = Interval(time.Duration(asInt64) * time.Nanosecond)
	return nil
}

func (i Interval) Value() (driver.Value, error) {
	return time.Duration(i).Nanoseconds(), nil
}

func (i Interval) IsZero() bool {
	return time.Duration(i) == time.Duration(0)
}

// WithdrawalRequest request to withdraw PHB.
type WithdrawalRequest struct {
	DestinationAddress common.Address `json:"address"`
	ContractAddress    common.Address `json:"contractAddress"`
	Amount             *assets.Phb   `json:"amount"`
}

type SendEtherRequest struct {
	DestinationAddress common.Address `json:"address"`
	FromAddress        common.Address `json:"from"`
	Amount             assets.Eth     `json:"amount"`
}

type AddressCollection []common.Address

func (r AddressCollection) ToStrings() []string {
	converted := make([]string, len(r))
	for i, e := range r {
		converted[i] = e.Hex()
	}
	return converted
}

func (r AddressCollection) Value() (driver.Value, error) {
	return strings.Join(r.ToStrings(), ","), nil
}

func (r *AddressCollection) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("unable to convert %v of %T to AddressCollection", value, value)
	}

	if len(str) == 0 {
		return nil
	}

	arr := strings.Split(str, ",")
	collection := make(AddressCollection, len(arr))
	for i, a := range arr {
		collection[i] = common.HexToAddress(a)
	}
	*r = collection
	return nil
}

type Configuration struct {
	ID        int64  `gorm:"primary_key"`
	Name      string `gorm:"not null;unique;index"`
	Value     string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *gorm.DeletedAt
}

func Merge(inputs ...JSON) (JSON, error) {
	output := make(map[string]interface{})

	for _, input := range inputs {
		switch v := input.Result.Value().(type) {
		case map[string]interface{}:
			for key, value := range v {
				output[key] = value
			}
		case nil:
		default:
			return JSON{}, errors.New("can only merge JSON objects")
		}
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return JSON{}, err
	}

	return JSON{Result: gjson.ParseBytes(bytes)}, nil
}

func MergeExceptResult(inputs ...JSON) (JSON, error) {
	output := make(map[string]interface{})

	for _, input := range inputs {
		switch v := input.Result.Value().(type) {
		case map[string]interface{}:
			for key, value := range v {
				if key == "result" {
					if _, exists := output["result"]; exists {
						// Do not overwrite result field
						continue
					}
				}
				output[key] = value
			}
		case nil:
		default:
			return JSON{}, errors.New("can only merge JSON objects")
		}
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return JSON{}, err
	}

	return JSON{Result: gjson.ParseBytes(bytes)}, nil
}

type Sha256Hash [32]byte

func (s Sha256Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Sha256Hash) UnmarshalJSON(input []byte) error {
	var shaHash string
	if err := json.Unmarshal(input, &shaHash); err != nil {
		return err
	}

	sha, err := Sha256HashFromHex(shaHash)
	if err != nil {
		return err
	}

	*s = sha
	return nil
}

func Sha256HashFromHex(x string) (Sha256Hash, error) {
	bs, err := hex.DecodeString(x)
	if err != nil {
		return Sha256Hash{}, err
	}
	var hash Sha256Hash
	copy(hash[:], bs)
	return hash, nil
}

func MustSha256HashFromHex(x string) Sha256Hash {
	bs, err := hex.DecodeString(x)
	if err != nil {
		panic(err)
	}
	var hash Sha256Hash
	copy(hash[:], bs)
	return hash
}

func (s Sha256Hash) String() string {
	return hex.EncodeToString(s[:])
}

func (s *Sha256Hash) UnmarshalText(bs []byte) error {
	x, err := hex.DecodeString(string(bs))
	if err != nil {
		return err
	}
	*s = Sha256Hash{}
	copy((*s)[:], x)
	return nil
}

func (s *Sha256Hash) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("Failed to unmarshal Sha256Hash value: %v", value)
	}
	if s == nil {
		*s = Sha256Hash{}
	}
	copy((*s)[:], bytes)
	return nil
}

func (s Sha256Hash) Value() (driver.Value, error) {
	b := make([]byte, 32)
	copy(b, s[:])
	return b, nil
}
