package confab_test

import (
	"errors"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func ContainSequence(expected interface{}) types.GomegaMatcher {
	return &containSequenceMatcher{
		expected: expected,
	}
}

type containSequenceMatcher struct {
	expected interface{}
}

func (matcher *containSequenceMatcher) Match(actual interface{}) (success bool, err error) {
	if reflect.TypeOf(actual).Kind() != reflect.Slice {
		return false, errors.New("not a slice")
	}

	expectedLength := reflect.ValueOf(matcher.expected).Len()
	actualLength := reflect.ValueOf(actual).Len()
	for i := 0; i < (actualLength - expectedLength + 1); i++ {
		match := reflect.ValueOf(actual).Slice(i, i+expectedLength)
		if reflect.DeepEqual(matcher.expected, match.Interface()) {
			return true, nil
		}
	}

	return false, nil
}

func (matcher *containSequenceMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain sequence", matcher.expected)
}

func (matcher *containSequenceMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain sequence", matcher.expected)
}

var _ = Describe("ContainSequence", func() {
	Context("when actual is not a slice", func() {
		It("should error", func() {
			_, err := ContainSequence(func() {}).Match("not a slice")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when actual is a slice", func() {
		var sequence []string

		BeforeEach(func() {
			sequence = []string{
				"value-1",
				"value-2",
				"value-3",
			}
		})

		Context("when the entire sequence is present", func() {
			It("should succeed", func() {
				Expect([]string{
					"value-0",
					"value-1",
					"value-2",
					"value-3",
					"value-4",
				}).To(ContainSequence(sequence))
			})
		})

		Context("when some of the sequence is present", func() {
			It("should fail", func() {
				Expect([]string{
					"value-0",
					"value-1",
					"value-3",
					"value-4",
				}).NotTo(ContainSequence(sequence))
			})
		})

		Context("when none of the sequence is present", func() {
			It("should fail", func() {
				Expect([]string{
					"value-0",
					"value-4",
				}).NotTo(ContainSequence(sequence))
			})
		})

		Context("when the elements match, but the order does not", func() {
			It("should fail", func() {
				Expect([]string{
					"value-0",
					"value-3",
					"value-1",
					"value-2",
					"value-4",
				}).NotTo(ContainSequence(sequence))
			})
		})

		Context("when the sequence shows up at the end of the actual slice", func() {
			It("should succeed", func() {
				Expect([]string{
					"value-0",
					"value-1",
					"value-2",
					"value-3",
				}).To(ContainSequence(sequence))
			})
		})
	})

	Describe("FailureMessage", func() {
		It("returns an understandable error message", func() {
			Expect(ContainSequence([]int{1, 2, 3}).FailureMessage([]int{5, 6})).To(Equal("Expected\n    <[]int | len:2, cap:2>: [5, 6]\nto contain sequence\n    <[]int | len:3, cap:3>: [1, 2, 3]"))
		})
	})

	Describe("NegatedFailureMessage", func() {
		It("returns an understandable error message", func() {
			Expect(ContainSequence([]int{1, 2, 3}).NegatedFailureMessage([]int{1, 2, 3})).To(Equal("Expected\n    <[]int | len:3, cap:3>: [1, 2, 3]\nnot to contain sequence\n    <[]int | len:3, cap:3>: [1, 2, 3]"))
		})
	})
})
