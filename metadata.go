package metadata

import (
	"errors"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnsupportedType  = errors.New("type is not supported")
	ErrConversion       = errors.New("couldn't convert value to requested type")
	ErrNotStructPointer = errors.New("passed data isn't a pointer to struct")
	ErrValueMissing     = errors.New("couldn't extract value from metadata")

	sliceOfStrings   = reflect.TypeOf([]string(nil))
	sliceOfBools     = reflect.TypeOf([]bool(nil))
	sliceOfInts      = reflect.TypeOf([]int(nil))
	sliceOfUints     = reflect.TypeOf([]uint(nil))
	sliceOfInt64s    = reflect.TypeOf([]int64(nil))
	sliceOfUint64s   = reflect.TypeOf([]uint64(nil))
	sliceOfFloat32s  = reflect.TypeOf([]float32(nil))
	sliceOfFloat64s  = reflect.TypeOf([]float64(nil))
	sliceOfDurations = reflect.TypeOf([]time.Duration(nil))
)

// TODO: Add Marshal/Unmarshal with interfaces support
type Readable interface {
	Labels() map[string]string
	Annotations() map[string]string
}

type Writeable interface {
	SetLabels(m map[string]string)
	SetAnnotations(m map[string]string)
}

func Unmarshal(metadata v1.ObjectMeta, data interface{}, prefix string) error {
	refPtr := reflect.ValueOf(data)
	if refPtr.Kind() != reflect.Ptr {
		return ErrNotStructPointer
	}

	ref := refPtr.Elem()
	if ref.Kind() != reflect.Struct {
		return ErrNotStructPointer
	}

	refType := ref.Type()

	for i := 0; i < refType.NumField(); i++ {
		refField := ref.Field(i)
		refTypeField := refType.Field(i)

		// Try to use annotations
		dict := metadata.Annotations
		tag, ok := refTypeField.Tag.Lookup("annotation")
		if !ok {
			// Fallback to using labels
			tag, ok = refTypeField.Tag.Lookup("label")
			if !ok {
				continue
			}

			dict = metadata.Labels
		}

		key := fmt.Sprintf("%s/%s", prefix, tag)
		value, ok := dict[key]
		if !ok {
			return ErrValueMissing
		}

		err := parseValue(refTypeField, refField, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func Marshal(metadata *v1.ObjectMeta, data interface{}, prefix string) error {
	refPtr := reflect.ValueOf(data)
	if refPtr.Kind() != reflect.Ptr {
		return ErrNotStructPointer
	}

	ref := refPtr.Elem()
	if ref.Kind() != reflect.Struct {
		return ErrNotStructPointer
	}

	refType := ref.Type()

	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}

	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}

	for i := 0; i < refType.NumField(); i++ {
		refField := ref.Field(i)
		refTypeField := refType.Field(i)

		dict := metadata.Annotations
		tag, ok := refTypeField.Tag.Lookup("annotation")
		if !ok {
			// Try to use labels
			tag, ok = refTypeField.Tag.Lookup("label")
			if !ok {
				continue
			}

			dict = metadata.Labels
		}

		key := fmt.Sprintf("%s/%s", prefix, tag)

		if refField.Kind() == reflect.Slice {
			var values []string
			for i := 0; i < refField.Len(); i++ {
				s := fmt.Sprint(refField.Index(i))
				values = append(values, s)
			}

			dict[key] = strings.Join(values, ",")
		} else {
			dict[key] = fmt.Sprint(refField)
		}
	}

	return nil
}

func parseValue(typeField reflect.StructField, valueField reflect.Value, value string) error {
	switch typeField.Type.Kind() {
	case reflect.String:
		valueField.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return ErrConversion
		}
		valueField.SetBool(b)
	case reflect.Int8:
		i, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			return ErrConversion
		}
		valueField.SetInt(i)
	case reflect.Int16:
		i, err := strconv.ParseInt(value, 10, 16)
		if err != nil {
			return ErrConversion
		}
		valueField.SetInt(i)
	case reflect.Int:
		fallthrough
	case reflect.Int32:
		i, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return ErrConversion
		}
		valueField.SetInt(i)
	case reflect.Int64:
		if typeField.Type.String() == "time.Duration" {
			d, err := time.ParseDuration(value)
			if err != nil {
				return ErrConversion
			}
			valueField.Set(reflect.ValueOf(d))
		} else {
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return ErrConversion
			}
			valueField.SetInt(i)
		}
	case reflect.Uint8:
		i, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return ErrConversion
		}
		valueField.SetUint(i)
	case reflect.Uint16:
		i, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return ErrConversion
		}
		valueField.SetUint(i)
	case reflect.Uint:
		fallthrough
	case reflect.Uint32:
		i, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return ErrConversion
		}
		valueField.SetUint(i)
	case reflect.Uint64:
		i, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return ErrConversion
		}
		valueField.SetUint(i)
	case reflect.Float32:
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return ErrConversion
		}
		valueField.SetFloat(f)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrConversion
		}
		valueField.SetFloat(f)
	case reflect.Complex64:
		c, err := strconv.ParseComplex(value, 64)
		if err != nil {
			return ErrConversion
		}
		valueField.SetComplex(c)
	case reflect.Complex128:
		c, err := strconv.ParseComplex(value, 128)
		if err != nil {
			return ErrConversion
		}
		valueField.SetComplex(c)
	case reflect.Slice:
		err := handleSlice(valueField, value, ",")
		if err != nil {
			return ErrConversion
		}
	default:
		return ErrUnsupportedType
	}
	return nil
}

func handleSlice(field reflect.Value, value, separator string) error {
	if separator == "" {
		separator = ","
	}

	splitData := strings.Split(value, separator)

	switch field.Type() {
	case sliceOfStrings:
		field.Set(reflect.ValueOf(splitData))
	case sliceOfBools:
		boolData, err := parseBools(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(boolData))
	case sliceOfInts:
		intData, err := parseInts(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(intData))
	case sliceOfUints:
		intData, err := parseUints(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(intData))
	case sliceOfInt64s:
		int64Data, err := parseInt64s(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(int64Data))
	case sliceOfUint64s:
		uint64Data, err := parseUint64s(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(uint64Data))
	case sliceOfFloat32s:
		data, err := parseFloat32s(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(data))
	case sliceOfFloat64s:
		data, err := parseFloat64s(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(data))
	case sliceOfDurations:
		durationData, err := parseDurations(splitData)
		if err != nil {
			return ErrConversion
		}
		field.Set(reflect.ValueOf(durationData))
	default:
		return ErrUnsupportedType
	}

	return nil
}

func parseBools(data []string) ([]bool, error) {
	boolSlice := make([]bool, 0, len(data))

	for _, v := range data {
		bValue, err := strconv.ParseBool(v)
		if err != nil {
			return nil, ErrConversion
		}

		boolSlice = append(boolSlice, bValue)
	}
	return boolSlice, nil
}

func parseInts(data []string) ([]int, error) {
	intSlice := make([]int, 0, len(data))

	for _, v := range data {
		intValue, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, ErrConversion
		}
		intSlice = append(intSlice, int(intValue))
	}
	return intSlice, nil
}

func parseUints(data []string) ([]uint, error) {
	uintSlice := make([]uint, 0, len(data))

	for _, v := range data {
		uintValue, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, ErrConversion
		}
		uintSlice = append(uintSlice, uint(uintValue))
	}
	return uintSlice, nil
}

func parseInt64s(data []string) ([]int64, error) {
	intSlice := make([]int64, 0, len(data))

	for _, v := range data {
		intValue, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, ErrConversion
		}
		intSlice = append(intSlice, intValue)
	}
	return intSlice, nil
}

func parseUint64s(data []string) ([]uint64, error) {
	var uintSlice []uint64

	for _, v := range data {
		uintValue, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, ErrConversion
		}
		uintSlice = append(uintSlice, uintValue)
	}
	return uintSlice, nil
}

func parseFloat32s(data []string) ([]float32, error) {
	float32Slice := make([]float32, 0, len(data))

	for _, v := range data {
		data, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, ErrConversion
		}
		float32Slice = append(float32Slice, float32(data))
	}
	return float32Slice, nil
}

func parseFloat64s(data []string) ([]float64, error) {
	float64Slice := make([]float64, 0, len(data))

	for _, v := range data {
		data, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, ErrConversion
		}
		float64Slice = append(float64Slice, data)
	}
	return float64Slice, nil
}

func parseDurations(data []string) ([]time.Duration, error) {
	durationSlice := make([]time.Duration, 0, len(data))

	for _, v := range data {
		dValue, err := time.ParseDuration(v)
		if err != nil {
			return nil, ErrConversion
		}

		durationSlice = append(durationSlice, dValue)
	}
	return durationSlice, nil
}
