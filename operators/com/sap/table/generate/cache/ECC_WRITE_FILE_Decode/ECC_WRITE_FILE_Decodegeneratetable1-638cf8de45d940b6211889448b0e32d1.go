package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	// Default, no-op input handler
	Trigger = func(_ interface{}) {}
	// Output
	Output func(interface{})
	// Configuration getters
	GetString func(string) string
	GetInt    func(string) int
	GetArray  func(string) []interface{}

	AddTimer func(time.Duration, int, func() error)

	staticTable   = map[string]interface{}{"version": 1}
	rowsPerOutput int
)

func Setup() error {
	rowsPerOutput = GetInt("rowsPerOutput")
	// Configuration mode
	mode := GetString("configMode")
	switch mode {
	case "Static (from configuration)":
		return staticSetup()
	case "Dynamic (from input)":
		Trigger = func(v interface{}) {
			generate(v.(map[string]interface{})["Attributes"].(map[string]interface{})["table"].(map[string]interface{}))
		}
		return nil
	default:
		return fmt.Errorf("unexpected value '%s' for property 'Configuration mode'", mode)
	}
}

func staticSetup() error {
	if err := setupStaticTable(); err != nil {
		return err
	}
	// Output mode
	mode := GetString("outputMode")
	switch mode {
	case "Once":
		AddTimer(time.Millisecond, 1, func() error {
			generate(staticTable)
			return nil
		})
	case "Periodic":
		period, err := time.ParseDuration(GetString("period"))
		if err != nil {
			return fmt.Errorf("failed to parse period: %s", err)
		}
		AddTimer(period, -1, func() error {
			generate(staticTable)
			return nil
		})
	case "Trigger":
		n := 0
		triggerCount := GetInt("triggerCount")
		Trigger = func(_ interface{}) {
			n++
			if n >= triggerCount {
				generate(staticTable)
				n = 0
			}
		}
	default:
		return fmt.Errorf("unexpected value '%s' for property 'Output mode'", mode)
	}
	return nil
}

func setupStaticTable() error {
	// Name
	if n := GetString("tableName"); n != "" {
		staticTable["name"] = n
	}
	// Columns
	configCols := GetArray("tableColumns")
	cols := make([]interface{}, len(configCols))
	for i, v := range configCols {
		cc := v.(map[string]interface{})
		c := map[string]interface{}{
			"name":     cc["name"],
			"nullable": cc["nullable"],
		}
		// Class and sizes
		if class, ok := cc["class"].(string); ok {
			class = strings.ToLower(class)
			c["class"] = class
			switch class {
			case "decimal":
				c["precision"] = cc["precision"]
				c["scale"] = cc["scale"]
			case "string":
				c["size"] = cc["size"]
			}
		}
		// Database-specific types
		if types, ok := cc["types"].([]interface{}); ok && len(types) > 0 {
			t := map[string]interface{}{}
			for _, v := range types {
				ct := v.(map[string]interface{})
				t[ct["database"].(string)] = ct["type"]
			}
			c["type"] = t
		}
		cols[i] = c
	}
	staticTable["columns"] = cols
	// Primary key
	if pk := GetArray("primaryKey"); len(pk) > 0 {
		if len(pk) == 1 {
			staticTable["primaryKey"] = pk[0]
		} else {
			staticTable["primaryKey"] = pk
		}
	}
	return nil
}

func generate(t map[string]interface{}) {
	gens := getGeneratorSet(t["columns"].([]interface{}))
	body := make([]interface{}, rowsPerOutput)
	for i := range body {
		body[i] = gens.generate()
	}
	Output(map[string]interface{}{
		"Attributes": map[string]interface{}{
			"table": t,
		},
		"Encoding": "table",
		"Body":     body,
	})
}

type generatorSet []func() interface{}

func (g generatorSet) generate() []interface{} {
	row := make([]interface{}, len(g))
	for i, gen := range g {
		row[i] = gen()
	}
	return row
}

func getGeneratorSet(cols []interface{}) generatorSet {
	gens := make(generatorSet, len(cols))
	for i, c := range cols {
		col := c.(map[string]interface{})
		switch col["class"].(string) {
		case "timestamp":
			gens[i] = generateTimestamp
		case "integer":
			gens[i] = generateInteger
		case "decimal":
			gens[i] = newDecimalGenerator(int(col["precision"].(float64)))
		case "float":
			gens[i] = generateFloat
		case "string":
			gens[i] = newStringGenerator(int(col["size"].(float64)))
		case "binary":
			gens[i] = generateBinary
		case "bool":
			gens[i] = generateBool
		}
	}
	return gens
}

func generateTimestamp() interface{} {
	return time.Now().Format(time.RFC3339Nano)
}

func generateInteger() interface{} {
	return rand.Int63n(1000)
}

func newDecimalGenerator(p int) func() interface{} {
	return func() interface{} {
		return math.Floor(rand.Float64() * math.Pow10(p))
	}
}

func generateFloat() interface{} {
	return rand.Float64()
}

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func newStringGenerator(s int) func() interface{} {
	return func() interface{} {
		out := make([]byte, s)
		for i := range out {
			out[i] = alphabet[rand.Intn(len(alphabet))]
		}
		return string(out)
	}
}

func generateBinary() interface{} {
	out := make([]byte, 10)
	for i := range out {
		out[i] = byte(rand.Intn(256))
	}
	return out
}

func generateBool() interface{} {
	return rand.Float32() < 0.5
}
