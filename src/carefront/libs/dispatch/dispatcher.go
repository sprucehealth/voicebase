/*
Package dispatch implements local pub-sub. (i.e. the observer pattern).

Example use:

	type MessageEvent struct {
		FromUserID int64
		ToUserID int64
		MessageID int64
	}

	d := NewDispatcher()

	// Register a listener for new message that sends a notification to the receiver
	d.Subscribe(func(e *MessageEvent) error {
		NotifyUser(e.ToUserID, "New message!", e.MessageID)
		return nil
	})

	// When a user sends a message, publish an event
	msgID := SendMessage(fromUserID, toUserID, "Hello World")
	d.Publish(&MessageEvent{
		FromUserID: fromUserID,
		ToUserID: toUserID,
		MessageID: msgID,
	})

Publish happens synchronously since there are many cases when it's important to
know that all listeners have processed the event before proceeding to provide
a consistent view of the world. Each listener then has the choice to have its
own asynchronous processing if it desires.

Errors are collected in the Publish, but generally it's not practical for the
caller to have any sane way to deal with the errors. Decoupling means that
it should not understand that those errors means.
*/
package dispatch

import (
	"fmt"
	"reflect"
	"strings"
)

type ErrorList []error

func (e ErrorList) Error() string {
	if len(e) == 1 {
		return fmt.Sprintf("dispatch: %s", e[0].Error())
	} else {
		s := make([]string, len(e))
		for i, err := range e {
			s[i] = err.Error()
		}
		return fmt.Sprintf("dispatch: %s", strings.Join(s, ","))
	}
}

type Dispatcher struct {
	listeners map[reflect.Type][]reflect.Value
}

var Default = New()

func New() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[reflect.Type][]reflect.Value),
	}
}

// Subscribe adds a listener for an event. The Listener must accept one argument that is
// either of type struct or pointer to a struct. The argument defines the type of values
// that the listener wants to receive. If the listener does not conform to this format
// then Subscribe will panic.
func (d *Dispatcher) Subscribe(l interface{}) {
	t := reflect.TypeOf(l)
	if t.NumIn() != 1 {
		panic("Dispatcher.Subscribe requires listener to accept exactly 1 argument")
	}
	in := t.In(0)
	if in.Kind() != reflect.Struct && in.Kind() != reflect.Ptr && in.Elem().Kind() != reflect.Struct {
		panic("Dispatcher.Subscribe requires the argument to listener to be a struct or pointer to a struct")
	}
	if t.NumOut() != 1 || t.Out(0).Name() != "error" {
		panic("Dispatcher.Subscribe requires listener to return exactly 1 value of type error")
	}
	d.listeners[in] = append(d.listeners[in], reflect.ValueOf(l))
}

// Publish synhronously delivers an event to all listeners that are looking for the type
// of the event. All listeners are executes regardless of any one returning an error. This
// is done since there is no explicit ordering for listeners. Publish will return nil if all
// listeners completed successfuly, otherwise it will return an error of type ErrorList
// that aggregates any and all errors.
func (d *Dispatcher) Publish(e interface{}) error {
	v := reflect.ValueOf(e)
	t := v.Type()
	args := []reflect.Value{v}
	var errors []error
	for _, l := range d.listeners[t] {
		if ev := l.Call(args)[0]; !ev.IsNil() {
			errors = append(errors, ev.Interface().(error))
		}
	}
	if len(errors) == 0 {
		return nil
	}
	return ErrorList(errors)
}
