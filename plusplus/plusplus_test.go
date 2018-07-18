package plusplus

import (
	"reflect"
	"testing"
)

var testCasesPP = []struct {
	name  string
	msg   string
	users []string
}{
	{
		name:  "no user mention or ++",
		msg:   "This is some text",
		users: []string{},
	},
	{
		name:  "no user mention with ++",
		msg:   "This is some text++",
		users: []string{},
	},
	{
		name:  "user mention without ++",
		msg:   "This is some text <@UBLKAG9K4>",
		users: []string{},
	},
	{
		name:  "single ++ mention",
		msg:   "This is some text <@UBLKAG9K4>++",
		users: []string{"UBLKAG9K4"},
	},
	{
		name:  "invalid user mention with ++",
		msg:   "This is an invalid user @Dave++",
		users: []string{},
	},
	{
		name:  "multiple user mentions with ++",
		msg:   "This is a double <@UBLKAG9K4>++ and <@UBLPTK0JH>++.",
		users: []string{"UBLKAG9K4", "UBLPTK0JH"},
	},
}

func TestIdentifyPlusPlus(t *testing.T) {
	for _, tc := range testCasesPP {
		t.Run(tc.name, func(st *testing.T) {
			users := identifyPlusPlus(tc.msg)

			if len(users) != len(tc.users) {
				t.Error("should return the correct number of users")
			}

			if len(tc.users) > 0 && reflect.DeepEqual(users, tc.users) == false {
				t.Error("should return the correct list of users")
			}
		})
	}
}

var testCasesMM = []struct {
	name  string
	msg   string
	users []string
}{
	{
		name:  "no user mention or --",
		msg:   "This is some text",
		users: []string{},
	},
	{
		name:  "no user mention with --",
		msg:   "This is some text--",
		users: []string{},
	},
	{
		name:  "user mention without --",
		msg:   "This is some text <@UBLKAG9K4>",
		users: []string{},
	},
	{
		name:  "single -- mention",
		msg:   "This is some text <@UBLKAG9K4>--",
		users: []string{"UBLKAG9K4"},
	},
	{
		name:  "invalid user mention with --",
		msg:   "This is an invalid user @Dave--",
		users: []string{},
	},
	{
		name:  "multiple user mentions with --",
		msg:   "This is a double <@UBLKAG9K4>-- and <@UBLPTK0JH>--.",
		users: []string{"UBLKAG9K4", "UBLPTK0JH"},
	},
}

func TestIdentifyMinusMinus(t *testing.T) {
	for _, tc := range testCasesMM {
		t.Run(tc.name, func(st *testing.T) {
			users := identifyMinusMinus(tc.msg)

			if len(users) != len(tc.users) {
				t.Error("should return the correct number of users")
			}

			if len(tc.users) > 0 && reflect.DeepEqual(users, tc.users) == false {
				t.Error("should return the correct list of users")
			}
		})
	}
}
