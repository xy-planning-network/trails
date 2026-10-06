package resp

import (
	"reflect"

	"github.com/go-json-experiment/jsonsplit"
	"github.com/xy-planning-network/trails/logger"
	"github.com/xy-planning-network/trails/postgres"
)

func newJSONsplit(l logger.Logger) *jsonsplit.Codec {
	c := &jsonsplit.Codec{
		AutoDetectOptions: true,
		ReportDifference: func(d jsonsplit.Difference) {
			t := d.GoType
			if tt, ok := d.GoValue.(jsonSchema); ok {
				if ttt, ok := tt.D.(postgres.PagedData); ok && ttt.Items != nil {
					t = reflect.TypeOf(ttt.Items)
					if t.Kind() == reflect.Slice {
						t = t.Elem()
					}
				} else if tt.D != nil {
					t = reflect.TypeOf(tt.D)
				}
			}

			var opts []string
			for name := range d.OptionNames() {
				opts = append(opts, name)
			}

			var tName string
			if t != nil {
				tName = t.Name()
			}

			l.Warn(
				"json-diff",
				&logger.LogContext{
					Data: map[string]any{
						"caller":  d.Caller,
						"func":    d.Func,
						"options": opts,
						"type":    tName,
					},
				},
			)
		},
	}

	c.SetMarshalCallMode(jsonsplit.CallBothButReturnV1)
	c.SetUnmarshalCallMode(jsonsplit.CallBothButReturnV1)
	return c
}
