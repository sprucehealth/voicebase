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

	"github.com/sprucehealth/backend/libs/golog"
)

// If Testing is set then PublishAsync is actually synchronous. This makes
// tests that rely on dispatch deterministic.
var Testing = false

//RunAsync runs the provided function in a go routine if Testing is not set,
// and synchronously if it is
func RunAsync(f func()) {
	if !Testing {
		go f()
	} else {
		f()
	}
}

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

type Publisher interface {
	Publish(e interface{}) error
}

type Dispatcher struct {
	listeners    map[reflect.Type][]subscriber
	allListeners []subscriber
}

type subscriber struct {
	async bool
	sub   reflect.Value
}

func New() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[reflect.Type][]subscriber),
	}
}

// Subscribe adds a listener for an event. The listener must accept one argument that is
// either of type struct or pointer to a struct. The argument defines the type of values
// that the listener wants to receive. If the listener does not conform to this format
// then Subscribe will panic.
func (d *Dispatcher) Subscribe(l interface{}) {
	d.subscribe(l, false, false)
}

func (d *Dispatcher) SubscribeAll(l interface{}) {
	d.subscribe(l, true, false)
}

// SubscribeAsync is the same as Subscribe, but it adds a listener that is executed in
// a separate goroutine.
func (d *Dispatcher) SubscribeAsync(l interface{}) {
	d.subscribe(l, false, true)
}

func (d *Dispatcher) SubscribeAllAsync(l interface{}) {
	d.subscribe(l, true, true)
}

func (d *Dispatcher) subscribe(l interface{}, all, async bool) {
	t := reflect.TypeOf(l)
	if t.NumIn() != 1 {
		panic("Dispatcher.Subscribe requires listener to accept exactly 1 argument")
	}
	in := t.In(0)
	if !all && in.Kind() != reflect.Struct && in.Kind() != reflect.Ptr && in.Elem().Kind() != reflect.Struct {
		panic("Dispatcher.Subscribe requires the argument to listener to be a struct or pointer to a struct")
	}
	if t.NumOut() != 1 || t.Out(0).Name() != "error" {
		panic("Dispatcher.Subscribe requires listener to return exactly 1 value of type error")
	}
	if all {
		d.allListeners = append(d.allListeners, subscriber{async: async, sub: reflect.ValueOf(l)})
	} else {
		d.listeners[in] = append(d.listeners[in], subscriber{async: async, sub: reflect.ValueOf(l)})
	}
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
		errors = d.notify(l, t, args, errors)
	}
	for _, l := range d.allListeners {
		errors = d.notify(l, t, args, errors)
	}
	if len(errors) == 0 {
		return nil
	}
	return ErrorList(errors)
}

func (d *Dispatcher) notify(s subscriber, t reflect.Type, args []reflect.Value, errors []error) []error {
	if !Testing && s.async {
		listener := s
		go func() {
			if ev := listener.sub.Call(args)[0]; !ev.IsNil() {
				e := ev.Interface().(error)
				golog.Errorf("Listener failed for type %+v: %s", t, e.Error())
			}
		}()
	} else {
		if ev := s.sub.Call(args)[0]; !ev.IsNil() {
			e := ev.Interface().(error)
			golog.Errorf("Listener failed for type %+v: %s", t, e.Error())
			errors = append(errors, e)
		}
	}
	return errors
}

// PublishAsync does the publishing in the background using a goroutine ignoring
// any errors returned by listeners.
func (d *Dispatcher) PublishAsync(e interface{}) {
	if Testing {
		d.Publish(e)
	} else {
		go d.Publish(e)
	}
}
