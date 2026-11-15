package app

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// cyclicDetector tracks the messages on the current path while a message
// descriptor is walked, so that recursive message types (trees, expressions,
// google.protobuf.Value) are not expanded forever.
//
// Only the path matters: the same message type may appear any number of
// times side by side (e.g. two fields of type B), only not inside itself.
type cyclicDetector struct {
	path map[protoreflect.FullName]struct{}
}

// enter records md as being walked. It returns false if md is already on the
// path, i.e. md (directly or indirectly) contains itself. Every successful
// enter must be followed by a leave.
func (d *cyclicDetector) enter(md protoreflect.MessageDescriptor) bool {
	if d.path == nil {
		d.path = make(map[protoreflect.FullName]struct{})
	}
	if _, ok := d.path[md.FullName()]; ok {
		return false
	}
	d.path[md.FullName()] = struct{}{}
	return true
}

func (d *cyclicDetector) leave(md protoreflect.MessageDescriptor) {
	delete(d.path, md.FullName())
}
