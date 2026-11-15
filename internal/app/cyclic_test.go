package app

import (
	"testing"

	"wombat/internal/server"
)

func fieldByName(t *testing.T, m *messageDesc, name string) fieldDesc {
	t.Helper()
	if m == nil {
		t.Fatalf("no message to look up %s in", name)
	}
	for _, f := range m.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("%s has no field %s", m.FullName, name)
	return fieldDesc{}
}

// Recursive message types must be described up to the point where they
// repeat, not fail and not recurse forever (a stack overflow kills the app).
func TestRecursiveMessageView(t *testing.T) {
	md := (&server.RecursiveRequest{}).ProtoReflect().Descriptor()
	m, err := messageViewFromDesc(md, &cyclicDetector{})
	if err != nil {
		t.Fatal(err)
	}

	// A message field before the recursive one used to reset the detector.
	if bar := fieldByName(t, m, "bar"); bar.Message == nil || bar.Message.Recursive || len(bar.Message.Fields) == 0 {
		t.Errorf("bar not expanded: %+v", bar.Message)
	}
	if self := fieldByName(t, m, "self"); self.Message == nil || !self.Message.Recursive || len(self.Message.Fields) != 0 {
		t.Errorf("self not marked recursive: %+v", self.Message)
	}

	tree := fieldByName(t, m, "tree")
	if tree.Message.Recursive {
		t.Fatal("tree marked recursive at its first occurrence")
	}
	if ch := fieldByName(t, tree.Message, "children"); !ch.Repeated || !ch.Message.Recursive {
		t.Errorf("tree.children not marked recursive: %+v", ch)
	}

	// The same type side by side (tree, then forest) is expanded both times.
	forest := fieldByName(t, m, "forest")
	if forest.MapValue == nil || forest.MapValue.Message == nil || forest.MapValue.Message.Recursive {
		t.Fatalf("forest value not expanded: %+v", forest.MapValue)
	}
	if ch := fieldByName(t, forest.MapValue.Message, "children"); !ch.Message.Recursive {
		t.Errorf("forest value children not marked recursive: %+v", ch)
	}
}

// Struct, Value and ListValue are edited as JSON, so they are not expanded
// (Value and ListValue contain each other).
func TestJSONMessageView(t *testing.T) {
	md := (&server.WellKnownRequest{}).ProtoReflect().Descriptor()
	m, err := messageViewFromDesc(md, &cyclicDetector{})
	if err != nil {
		t.Fatal(err)
	}
	for name, full := range map[string]string{
		"struct_value": "google.protobuf.Struct",
		"value":        "google.protobuf.Value",
		"list_value":   "google.protobuf.ListValue",
	} {
		f := fieldByName(t, m, name)
		if f.Message == nil || f.Message.FullName != full || f.Message.Recursive {
			t.Errorf("%s: got %+v, want %s", name, f.Message, full)
			continue
		}
		if len(f.Message.Fields) != 1 || f.Message.Fields[0].Kind != "string" {
			t.Errorf("%s: fields %+v, want a single string", name, f.Message.Fields)
		}
	}
}

// Selecting a method with a recursive request works and is what the
// frontend gets.
func TestSelectRecursiveMethod(t *testing.T) {
	a, rec := newConnectedAPI(t)
	if err := a.SelectMethod("/wombat.v1.Foobar/Recursive", "", nil); err != nil {
		t.Fatal(err)
	}
	evs := rec.all(eventMethodInputChanged)
	in := evs[len(evs)-1].data[0].(methodInput)
	if !fieldByName(t, in.Message, "self").Message.Recursive {
		t.Error("self not marked recursive")
	}
}
