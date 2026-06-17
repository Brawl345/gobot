package gpt

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Definition() FunctionTool {
	return FunctionTool{
		Type:        "function",
		Name:        "calculator",
		Description: "Führt präzise mathematische Berechnungen durch. Unterstützt +, -, *, /, ^ (Potenz), Klammern und Funktionen: sqrt, abs, floor, ceil, round, log (Basis 10), ln, sin, cos, tan, asin, acos, atan, pi, e.",
		Parameters: FunctionParameters{
			Type: "object",
			Properties: map[string]Property{
				"expression": {
					Type:        "string",
					Description: "Der mathematische Ausdruck, z.B. \"2^10\", \"sqrt(144)\", \"sin(pi/2)\"",
				},
			},
			Required:             []string{"expression"},
			AdditionalProperties: false,
		},
		Strict: true,
	}
}

func (t *CalculatorTool) Execute(arguments string) (any, error) {
	var args struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	log.Debug().Str("expression", args.Expression).Msg("calculator tool call")

	result, err := evalExpression(args.Expression)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	if math.IsInf(result, 0) {
		return "Error: division by zero or overflow", nil
	}
	if math.IsNaN(result) {
		return "Error: invalid mathematical operation (NaN)", nil
	}

	if result == math.Trunc(result) && math.Abs(result) < 1e15 {
		return fmt.Sprintf("%g", result), nil
	}
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func (t *CalculatorTool) Emoji() string {
	return "🧮"
}

const maxParseDepth = 500

// parser holds state for recursive-descent parsing.
type parser struct {
	input string
	pos   int
	depth int
}

func evalExpression(expr string) (float64, error) {
	p := &parser{input: strings.TrimSpace(expr)}
	result, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character: %q", string(p.input[p.pos]))
	}
	return result, nil
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *parser) peek() (byte, bool) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return 0, false
	}
	return p.input[p.pos], true
}

// parseExpr handles + and - (lowest precedence).
func (p *parser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		ch, ok := p.peek()
		if !ok || (ch != '+' && ch != '-') {
			break
		}
		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if ch == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

// parseTerm handles * and /.
func (p *parser) parseTerm() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for {
		ch, ok := p.peek()
		if !ok || (ch != '*' && ch != '/') {
			break
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		if ch == '*' {
			left *= right
		} else {
			left /= right
		}
	}
	return left, nil
}

// parseUnary handles unary signs. It binds weaker than ^ so that
// -2^2 == -(2^2) == -4, matching mathematical convention.
func (p *parser) parseUnary() (float64, error) {
	neg := false
	for {
		ch, ok := p.peek()
		if !ok || (ch != '-' && ch != '+') {
			break
		}
		if ch == '-' {
			neg = !neg
		}
		p.pos++
	}
	val, err := p.parsePower()
	if neg {
		val = -val
	}
	return val, err
}

// parsePower handles ^ (right-associative). Every recursion cycle in the
// parser passes through here, so this is where the depth limit lives.
func (p *parser) parsePower() (float64, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxParseDepth {
		return 0, fmt.Errorf("expression too deeply nested")
	}
	base, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}
	ch, ok := p.peek()
	if !ok || ch != '^' {
		return base, nil
	}
	p.pos++
	exp, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	return math.Pow(base, exp), nil
}

// parsePrimary handles numbers, parentheses, and named functions/constants.
func (p *parser) parsePrimary() (float64, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	ch := p.input[p.pos]

	if ch == '(' {
		p.pos++
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return val, nil
	}

	if unicode.IsLetter(rune(ch)) || ch == '_' {
		return p.parseIdentifier()
	}

	if unicode.IsDigit(rune(ch)) || ch == '.' {
		return p.parseNumber()
	}

	return 0, fmt.Errorf("unexpected character: %q", string(ch))
}

func (p *parser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '.') {
		p.pos++
	}
	if p.pos < len(p.input) && (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
			p.pos++
		}
		for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
			p.pos++
		}
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

func (p *parser) parseIdentifier() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '_') {
		p.pos++
	}
	name := p.input[start:p.pos]

	// Constants
	switch strings.ToLower(name) {
	case "pi":
		return math.Pi, nil
	case "e":
		return math.E, nil
	case "phi":
		return math.Phi, nil
	case "inf", "infinity":
		return math.Inf(1), nil
	}

	// Functions — require parenthesized argument
	p.skipWhitespace()
	if p.pos >= len(p.input) || p.input[p.pos] != '(' {
		return 0, fmt.Errorf("unknown constant or function: %q", name)
	}
	p.pos++ // consume '('
	arg, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipWhitespace()
	if p.pos >= len(p.input) || p.input[p.pos] != ')' {
		return 0, fmt.Errorf("missing closing parenthesis after %q", name)
	}
	p.pos++

	switch strings.ToLower(name) {
	case "sqrt":
		if arg < 0 {
			return 0, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(arg), nil
	case "abs":
		return math.Abs(arg), nil
	case "floor":
		return math.Floor(arg), nil
	case "ceil":
		return math.Ceil(arg), nil
	case "round":
		return math.Round(arg), nil
	case "log":
		if arg <= 0 {
			return 0, fmt.Errorf("log of non-positive number")
		}
		return math.Log10(arg), nil
	case "ln":
		if arg <= 0 {
			return 0, fmt.Errorf("ln of non-positive number")
		}
		return math.Log(arg), nil
	case "sin":
		return math.Sin(arg), nil
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "asin":
		return math.Asin(arg), nil
	case "acos":
		return math.Acos(arg), nil
	case "atan":
		return math.Atan(arg), nil
	case "exp":
		return math.Exp(arg), nil
	case "sign", "sgn":
		if arg == 0 {
			return 0, nil
		}
		return math.Copysign(1, arg), nil
	case "trunc":
		return math.Trunc(arg), nil
	}

	return 0, fmt.Errorf("unknown function: %q", name)
}
