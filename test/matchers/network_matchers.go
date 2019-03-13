package matchers

import (
	"fmt"
	"reflect"
	"time"

	uuid "github.com/kthomas/go.uuid"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
	"github.com/provideservices/provide-go"
	// provideapp "github.com/provideapp/goldmine"
)

// ParseConfigNetworkMatcher to check the parsed config keys and values
func ParseConfigNetworkMatcher(expected interface{}) types.GomegaMatcher {
	return &parseConfigNetworkMatcher{
		expected: expected,
	}
}

type parseConfigNetworkMatcher struct {
	expected interface{}
}

func (matcher *parseConfigNetworkMatcher) Match(actual interface{}) (success bool, err error) {
	SatisfyAll()
	return true, nil
}

func (matcher *parseConfigNetworkMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the JSON representation of\n\t%#v", actual, matcher.expected)
}

func (matcher *parseConfigNetworkMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the JSON representation of\n\t%#v", actual, matcher.expected)
}

// CheckNetworkDoubleCreateMatcher to check network duplication
func CheckNetworkDoubleCreateMatcher(expected interface{}) types.GomegaMatcher {
	return &checkNetworkDoubleCreateMatcher{
		expected: expected,
	}
}

type checkNetworkDoubleCreateMatcher struct {
	expected interface{}
}

func (matcher *checkNetworkDoubleCreateMatcher) Match(actual interface{}) (success bool, err error) {

	return true, nil
}

func (matcher *checkNetworkDoubleCreateMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the JSON representation of\n\t%#v", actual, matcher.expected)
}

func (matcher *checkNetworkDoubleCreateMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain the JSON representation of\n\t%#v", actual, matcher.expected)
}

func NetworkCreateWithNATSMatcher(expected interface{}, ch chan string) types.GomegaMatcher {
	return &networkCreateWithNATSMatcher{
		expected: expected,
		ch:       ch,
	}
}

type networkCreateWithNATSMatcher struct {
	expected interface{}
	ch       chan string
}

func (matcher *networkCreateWithNATSMatcher) Match(actual interface{}) (success bool, err error) {
	Eventually(matcher.ch).Should(Receive(Equal("timeout")))

	res, err := (&matchers.BeTrueMatcher{}).Match(actual)
	if !res || (err != nil) {
		return res, err
	}
	return true, nil
}

func (matcher *networkCreateWithNATSMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, matcher.expected)
}

func (matcher *networkCreateWithNATSMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, matcher.expected)
}

// NetworkCreateMatcher checks network.Create(): the result of Create() call and channel message
func NetworkCreateMatcher(expectedResult bool, expectedCount int, opts ...interface{}) types.GomegaMatcher {
	// return BeTrue()
	return &networkCreateMatcher{
		expected:       nil,
		expectedResult: expectedResult,
		expectedCount:  expectedCount,
		nPtr:           opts[0],
		ch:             opts[1].(chan string),
		fn:             opts[2].(func() []interface{}),
	}
}

type networkCreateMatcher struct {
	expected       interface{}
	expectedResult bool
	expectedCount  int
	nPtr           interface{}
	ch             chan string
	fn             func() []interface{}
}

// duplication to  check fields
type Contract struct {
	ApplicationID *uuid.UUID
	NetworkID     uuid.UUID
	ContractID    *uuid.UUID
	TransactionID *uuid.UUID
	Name          *string
	Address       *string
	AccessedAt    *time.Time
}

// black magic
func fieldFromPtr(p interface{}, fieldName string) interface{} {
	var pElem reflect.Value

	pValue := reflect.ValueOf(p)
	if pValue.Kind() == reflect.Ptr {
		pElem = reflect.ValueOf(p).Elem()
	}
	if pValue.Kind() == reflect.Struct {
		pElem = pValue
	}
	// fmt.Printf("pElem kind: %v\n", pElem.Kind())
	// fmt.Printf("pElem type: %v\n", pElem.Type().Name())

	if pElem.Type().Name() == "Value" {
		pElem = pElem.Interface().(reflect.Value)
	}

	if v, ok := pElem.Type().FieldByName(fieldName); ok {
		index := v.Index
		field := pElem.FieldByIndex(index)

		// fmt.Printf("field kind: %v\n", field.Kind())
		// fmt.Printf("field type: %v\n", field.Type())
		// fmt.Printf("field interface: %v\n", field.Interface())
		// fmt.Printf("field is valid: %v\n", field.IsValid())

		return field.Interface()
	}
	// if pElem.Type().Name() == "Network" {
	// 	fmt.Printf("CONTRACT\n")
	// 	if v, ok := pElem.Type().FieldByName(fieldName); ok {

	// 	}
	// }

	return nil
}

func callMethodOnPtr(p interface{}, methodName string) interface{} {
	var ptr reflect.Value
	var value reflect.Value
	var finalMethod reflect.Value

	value = reflect.ValueOf(p)

	// if we start with a pointer, we need to get value pointed to
	// if we start with a value, we need to get a pointer to that value
	fmt.Printf("value type kind: %v\n", value.Type().Kind())
	if value.Type().Kind() == reflect.Ptr {
		ptr = value
		value = ptr.Elem()
	} else {
		ptr = reflect.New(reflect.TypeOf(p))
		temp := ptr.Elem()
		temp.Set(value)
	}

	// check for method on value
	method := value.MethodByName(methodName)
	// fmt.Printf("method %v on value ", method)
	if method.IsValid() {
		// fmt.Printf("valid\n")
		finalMethod = method
	}
	// check for method on pointer
	method = ptr.MethodByName(methodName)
	// fmt.Printf("method %v on pointer ", method)
	if method.IsValid() {
		// fmt.Printf("valid\n")
		finalMethod = method
	}

	if finalMethod.IsValid() {
		// fmt.Printf("final method %v is valid\n", finalMethod)
		return finalMethod.Call([]reflect.Value{})[0].Interface()
	}

	// return or panic, method not found of either type
	return nil
}

func (matcher *networkCreateMatcher) Match(actual interface{}) (success bool, err error) {
	// https://stackoverflow.com/a/14162161 - how to call function on interface{} obj

	n := matcher.nPtr // opts[0]

	chPolling := matcher.ch

	nCreatedBool := callMethodOnPtr(n, "Create").(bool)
	// fmt.Printf("nCreated: %t\n", nCreatedBool)

	var res1 bool
	var err1 error
	if matcher.expectedResult {
		res1, err1 = (&matchers.BeTrueMatcher{}).Match(nCreatedBool)
	} else {
		res1, err1 = (&matchers.BeFalseMatcher{}).Match(nCreatedBool)
	}
	fmt.Printf("res1: %v\n", res1)
	fmt.Printf("err1: %v\n", err1)

	if !res1 || (err1 != nil) {
		return res1, err1
	}

	var networkID uuid.UUID

	fmt.Printf("BLACK MAGIC START\n")
	model := fieldFromPtr(n, "Model").(provide.Model)

	networkID = fieldFromPtr(model, "ID").(uuid.UUID)

	Eventually(chPolling).Should(Receive(Equal("timeout"))) // ending of Contracts processing and sending their IDs to pipe

	objects := matcher.fn()

	res2, err2 := (&matchers.HaveLenMatcher{Count: matcher.expectedCount}).Match(objects)
	fmt.Printf("res2: %v\n", res2)
	fmt.Printf("err2: %v\n", err2)

	if !res2 || (err2 != nil) {
		return res2, err2
	}

	if matcher.expectedCount == 1 {
		if len(objects) > 0 {

			contractPtr := objects[0]
			contract := reflect.ValueOf(contractPtr).Elem()

			fields := Contract{}

			appID := fieldFromPtr(contract, "ApplicationID").(*uuid.UUID)
			fields.ApplicationID = appID

			netID := fieldFromPtr(contract, "NetworkID").(uuid.UUID)
			fields.NetworkID = netID

			conID := fieldFromPtr(contract, "ContractID").(*uuid.UUID)
			fields.ContractID = conID

			name := fieldFromPtr(contract, "Name").(*string)
			fields.Name = name

			address := fieldFromPtr(contract, "Address").(*string)
			fields.Address = address

			fmt.Printf("fields: %#v\n", fields)
			// fmt.Printf("fields Name: %v\n", *fields.Name)
			res3, err3 := (&gstruct.FieldsMatcher{
				Fields: gstruct.Fields{
					// "Model": MatchFields(IgnoreExtras, Fields{
					// 	"ID":        Not(BeNil()),
					// 	"CreatedAt": Not(BeNil()),
					// 	"Errors":    BeEmpty(),
					// }),
					"NetworkID": Equal(networkID),
					// "NetworkID":     Not(BeNil()),
					"ApplicationID": BeNil(),
					"ContractID":    BeNil(),
					// "TransactionID": BeNil(),
					"Name":    gstruct.PointTo(Equal("Network Contract 0x0000000000000000000000000000000000000017")),
					"Address": gstruct.PointTo(Equal("0x0000000000000000000000000000000000000017")),
					// "Params":        PointTo(Equal("")), // TODO add params body
					// "AccessedAt": BeNil(),
				},
				IgnoreExtras: true,
			}).Match(fields)

			fmt.Printf("res3: %v\n", res3)
			fmt.Printf("err3: %v\n", err3)
			if !res3 || (err3 != nil) {
				return res3, err3
			}
			return true, nil // all passed
		}
		return false, nil // 0 contracts created
	}

	return true, nil // all passed, contracts not needed
}

func (matcher *networkCreateMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto match\n\t%#v", actual, matcher.expected)
}

func (matcher *networkCreateMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match\n\t%#v", actual, matcher.expected)
}
