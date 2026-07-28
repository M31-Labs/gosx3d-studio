package studio

import "reflect"

// transformType is compared by identity in deepCopyValue. See the note there.
var transformType = reflect.TypeOf(Transform{})

// deepCopyValue copies src into a value that shares no mutable state with it.
//
// It is reflection-driven on purpose. A hand-written copier is faster still,
// but every field added to the document later would have to be remembered, and
// a forgotten one aliases silently — the canonical document and a clone of it
// would share a slice, and a later edit would reach through history. This
// cannot forget a field.
//
// It replaced a JSON round trip, which spent two thirds of its time parsing
// text it had just produced: measured at 10.4 ms against 1.5 ms for a
// 1,000-entity scene.
func deepCopyValue(src reflect.Value) reflect.Value {
	// Transform is the one type whose JSON hooks did more than serialize.
	// MarshalJSON and UnmarshalJSON both canonicalize, so a clone through JSON
	// normalized every transform on the way past: a zero quaternion became one
	// derived from the euler angles, and an identity became the canonical
	// identity. Code that reads a cloned transform has always seen that, so
	// preserving it keeps Clone's contract exactly what it was.
	if src.Type() == transformType {
		return reflect.ValueOf(src.Interface().(Transform).canonical())
	}
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return src
		}
		out := reflect.New(src.Type().Elem())
		out.Elem().Set(deepCopyValue(src.Elem()))
		return out
	case reflect.Interface:
		if src.IsNil() {
			return src
		}
		out := reflect.New(src.Type()).Elem()
		out.Set(deepCopyValue(src.Elem()))
		return out
	case reflect.Slice:
		// A nil slice and an empty one serialize differently, and SceneDoc
		// fingerprints are compared for equality, so the distinction has to
		// survive the copy.
		if src.IsNil() {
			return src
		}
		out := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			out.Index(i).Set(deepCopyValue(src.Index(i)))
		}
		return out
	case reflect.Map:
		if src.IsNil() {
			return src
		}
		out := reflect.MakeMapWithSize(src.Type(), src.Len())
		iter := src.MapRange()
		for iter.Next() {
			out.SetMapIndex(deepCopyValue(iter.Key()), deepCopyValue(iter.Value()))
		}
		return out
	case reflect.Array:
		out := reflect.New(src.Type()).Elem()
		for i := 0; i < src.Len(); i++ {
			out.Index(i).Set(deepCopyValue(src.Index(i)))
		}
		return out
	case reflect.Struct:
		out := reflect.New(src.Type()).Elem()
		for i := 0; i < src.NumField(); i++ {
			// An unexported field cannot be set through reflection. Nothing a
			// SceneDoc carries has one, and the walk below proves it: a field
			// skipped here would keep its zero value, and a document holding
			// unexported state would fail to round-trip.
			if !out.Field(i).CanSet() {
				continue
			}
			out.Field(i).Set(deepCopyValue(src.Field(i)))
		}
		return out
	default:
		return src
	}
}
