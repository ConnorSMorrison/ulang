package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
)

type Vars struct {
	Vars      map[string]float64
	Functions map[string]*FunctionStatement
}

func (v *Vars) Create() {
	v.Vars = make(map[string]float64)
	v.Functions = make(map[string]*FunctionStatement)
}

var vars = &Vars{}
var inFunction = false

type StatementList struct {
	Statements []*Statement `@@*`
}

type Statement struct {
	Assign   *AssignStatement   `@@`
	If       *IfStatement       `| @@`
	While    *WhileStatement    `| @@`
	Until    *UntilStatement    `| @@`
	Print    *PrintStatement    `| @@`
	Input    *InputStatement    `| @@`
	Function *FunctionStatement `| @@`
	FunCall  *FunCallStatement  `| @@`
	Return   *ReturnStatement   `| @@`
}

type AssignStatement struct {
	Ident *string `@Ident `
	Expr  *Add    `"=" @@`
}

type IfStatement struct {
	Condition *Condition     `"if" @@ "then"`
	Body      *StatementList `@@`
	ElseIf    []*ElseIf      `@@*`
	Else      *Else          `@@?`
	End       string         `"end"`
}

type ElseIf struct {
	Condition *Condition     `"elseif" @@ "then"`
	Body      *StatementList `@@`
}

type Else struct {
	Body *StatementList `"else" @@`
}

type WhileStatement struct {
	Condition *Condition     `"while" @@ "do"`
	Body      *StatementList `@@ "end"`
}

type UntilStatement struct {
	Condition *Condition     `"until" @@ "do"`
	Body      *StatementList `@@ "end"`
}

type PrintStatement struct {
	Expr1 *AddList `"print" @@`
}

type InputStatement struct {
	Ident *string `"input" @Ident`
}

type FunctionStatement struct {
	Ident      *string        `"func" @Ident`
	Parameters *Parameters    `"(" @@ ")"`
	Body       *StatementList `@@ "end"`
}

type FunCallStatement struct {
	Ident     *string    `@Ident`
	Arguments *Arguments `"(" @@ ")"`
}

type ReturnStatement struct {
	Expr *Add `"return" @@`
}

type Parameters struct {
	Params []string `( @Ident ("," @Ident)* )?`
}

type Arguments struct {
	Args []Add `( @@ ("," @@)* )?`
}

type AddList struct {
	List []Add `( @@ ("," @@)* )`
}

type Condition struct {
	Add1     *Add       `@@`
	Cond     *string    `@("=" "="|">"|"<"|"!" "="|"<" "="|">" "=")`
	Add2     *Add       `@@`
	Logic    *string    `[ @("and"|"or")`
	MoreCond *Condition `@@ ]`
}

type Add struct {
	Mul *Mul    `@@`
	Op  *string `[ @("+"|"-")`
	Add *Add    `@@ ]`
}

type Mul struct {
	Value *Value  `@@`
	Op    *string `[ @("*"|"/")`
	Mul   *Mul    `@@ ]`
}

type Value struct {
	Unary            *string           `@"-"?`
	Number           *float64          `( @(Int|Float)`
	FunCallStatement *FunCallStatement `| @@`
	Variable         *string           `| @Ident`
	Parens           *Add              `| "(" @@ ")" )`
	Exp              *float64          `( "^" @(Int|Float))?`
}

func evalStatementList(list *StatementList, scope *Vars) *float64 {
	var val *float64
	for i := 0; i < len(list.Statements); i += 1 {
		val = evalStatement(list.Statements[i], scope)
		if val != nil {
			return val
		}
	}

	var n float64 = 0
	return &n
}

func evalStatement(stmt *Statement, scope *Vars) *float64 {
	var val *float64 = nil
	if stmt.Assign != nil {
		evalAssignStatement(stmt.Assign, scope)
	} else if stmt.If != nil {
		evalIfStatement(stmt.If, scope)
	} else if stmt.While != nil {
		evalWhileStatement(stmt.While, scope)
	} else if stmt.Until != nil {
		evalUntilStatement(stmt.Until, scope)
	} else if stmt.Print != nil {
		evalPrintStatement(stmt.Print, scope)
	} else if stmt.Input != nil {
		evalInputStatement(stmt.Input, scope)
	} else if stmt.Function != nil {
		evalFunctionStatement(stmt.Function)
	} else if stmt.FunCall != nil {
		evalFunctionCall(stmt.FunCall)
	} else if stmt.Return != nil {
		a := evalReturnStatement(stmt.Return, scope)
		val = &a
	}

	return val
}

func evalAssignStatement(aStmt *AssignStatement, scope *Vars) {
	scope.Vars[*aStmt.Ident] = evalAdd(aStmt.Expr, scope)
}

func evalIfStatement(iStmt *IfStatement, scope *Vars) {
	cond := evalCondition(iStmt.Condition, scope)
	if cond == true {
		evalStatementList(iStmt.Body, scope)
		return
	} else {
		if len(iStmt.ElseIf) != 0 {
			for i := 0; i < len(iStmt.ElseIf); i += 1 {
				elseif := evalElseIf(iStmt.ElseIf[i], scope)
				if elseif == true {
					return
				}
			}
		}

		if iStmt.Else != nil {
			evalElse(iStmt.Else, scope)
		}
	}
}

func evalElseIf(eiStmt *ElseIf, scope *Vars) bool {
	if evalCondition(eiStmt.Condition, scope) == true {
		evalStatementList(eiStmt.Body, scope)
		return true
	}
	return false
}

func evalElse(eStmt *Else, scope *Vars) {
	evalStatementList(eStmt.Body, scope)
}

func evalWhileStatement(wStmt *WhileStatement, scope *Vars) {
	for evalCondition(wStmt.Condition, scope) == true {
		evalStatementList(wStmt.Body, scope)
	}
}

func evalUntilStatement(uStmt *UntilStatement, scope *Vars) {
	for !(evalCondition(uStmt.Condition, scope) == true) {
		evalStatementList(uStmt.Body, scope)
	}
}

func evalPrintStatement(pStmt *PrintStatement, scope *Vars) {
	if len(pStmt.Expr1.List) > 1 {
		for i := 0; i < len(pStmt.Expr1.List); i += 1 {
			add := *&pStmt.Expr1.List[i]
			fmt.Print(evalAdd(&add, scope), " ")
		}
	} else {
		add := *&pStmt.Expr1.List[0]
		fmt.Print(evalAdd(&add, scope))
	}

	fmt.Println()
}

func evalInputStatement(iStmt *InputStatement, scope *Vars) {
	in := bufio.NewReader(os.Stdin)
	value, err := in.ReadString('\n')
	value = strings.TrimSpace(value)
	if err != nil {
		panic(err)
	}
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}

	scope.Vars[*iStmt.Ident] = floatVal
}

func evalFunctionStatement(fStmt *FunctionStatement) {
	vars.Functions[*fStmt.Ident] = fStmt
}

func evalFunctionCall(fCall *FunCallStatement) *float64 {
	scope := &Vars{}
	scope.Create()
	var returned *float64
	if _, ok := vars.Functions[*fCall.Ident]; ok {
		parameters := vars.Functions[*fCall.Ident].Parameters.Params
		if len(parameters) != len(fCall.Arguments.Args) {
			panic(fmt.Sprintf("Number of arguments does not match number of parameters of function '%s'. Wanted %d, got %d", *fCall.Ident, len(parameters), len(fCall.Arguments.Args)))
		}

		for i := 0; i < len(vars.Functions[*fCall.Ident].Parameters.Params); i += 1 {
			add := fCall.Arguments.Args[i]
			val := evalAdd(&add, scope)
			scope.Vars[vars.Functions[*fCall.Ident].Parameters.Params[i]] = val
		}

		inFunction = true
		returned = evalStatementList(vars.Functions[*fCall.Ident].Body, scope)
		inFunction = false
	} else {
		panic(fmt.Sprintf("Name '%s' not a function", *fCall.Ident))
	}

	var val *float64

	if returned != nil {
		val = returned
	}

	return val
}

func evalReturnStatement(rStmt *ReturnStatement, scope *Vars) float64 {
	if inFunction == true {
		val := evalAdd(rStmt.Expr, scope)
		return val
	} else {
		panic("Return cannot be used outside of a function")
	}
}

func evalCondition(condition *Condition, scope *Vars) bool {
	add1 := evalAdd(condition.Add1, scope)
	add2 := evalAdd(condition.Add2, scope)
	cond := *condition.Cond
	var val bool
	if cond == "==" {
		val = add1 == add2
	} else if cond == "!=" {
		val = add1 != add2
	} else if cond == ">" {
		val = add1 > add2
	} else if cond == "<" {
		val = add1 < add2
	} else if cond == ">=" {
		val = add1 >= add2
	} else if cond == "<=" {
		val = add1 <= add2
	}

	if condition.Logic != nil {
		val2 := evalCondition(condition.MoreCond, scope)

		if *condition.Logic == "and" {
			return val2 && val
		} else if *condition.Logic == "or" {
			return val2 || val
		}
	} else {
		return val
	}

	return false
}

func evalAdd(add *Add, scope *Vars) float64 {
	mul := evalMul(add.Mul, scope)

	if add.Op != nil {
		if *add.Op == "+" {
			mul += evalAdd(add.Add, scope)
		} else if *add.Op == "/" {
			mul -= evalAdd(add.Add, scope)
		}
	}

	return mul
}

func evalMul(mul *Mul, scope *Vars) float64 {
	val := evalValue(mul.Value, scope)

	if mul.Op != nil {
		if *mul.Op == "*" {
			val *= evalMul(mul.Mul, scope)
		} else if *mul.Op == "/" {
			val /= evalMul(mul.Mul, scope)
		}
	}

	return val
}

func evalValue(value *Value, scope *Vars) float64 {
	var val float64
	if value.Number != nil {
		val = *value.Number
	} else if value.Variable != nil {
		if _, ok := scope.Vars[*value.Variable]; ok {
			val = scope.Vars[*value.Variable]
		} else {
			if _, ok := vars.Vars[*value.Variable]; ok {
				val = vars.Vars[*value.Variable]
			} else {
				panic(fmt.Sprintf("Name '%s' not in variables", *value.Variable))
			}
		}
	} else if value.Parens != nil {
		new_val := evalAdd(value.Parens, scope)
		val = new_val
	} else if value.FunCallStatement != nil {
		new_val := evalFunctionCall(value.FunCallStatement)
		val = *new_val
	}

	if value.Exp != nil {
		val = math.Pow(val, *value.Exp)
	}

	if value.Unary != nil {
		val = -val
	}

	return val
}

func main() {
	vars.Create()

	input := `
if 1 > 2 and 3 < 4 then
	print 4
elseif 2 < 2 then
	print 2
else
	print 1
end
`

	parser := participle.MustBuild(&StatementList{})
	expr := &StatementList{}
	err := parser.ParseString("", input, expr)

	if err != nil {
		panic(err)
	}

	evalStatementList(expr, vars)
}
