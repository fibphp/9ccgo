package main

// util.go

// Vector
type Vector struct {
	data     []interface{}
	capacity int
	len      int
}

// Map
type Map struct {
	keys *Vector
	vals *Vector
}

// StringBuilder
type StringBuilder struct {
	data     string
	capacity int
	len      int
}

type Type struct {
	ty    int
	size  int // sizeof
	align int // alignof

	// Pointer
	ptr_to *Type

	// Array
	ary_of *Type
	len    int

	// Struct
	members *Vector
	offset  int

	// Function
	returning *Type
}

// token.go

const TK_NUM = 256      // Number literal
const TK_STR = 257      // String literal
const TK_IDENT = 258    // Identifier
const TK_ARROW = 259    // ->
const TK_EXTERN = 260   // "extern"
const TK_TYPEDEF = 261  // "typedef"
const TK_INT = 262      // "int"
const TK_CHAR = 263     // "char"
const TK_VOID = 264     // "void"
const TK_STRUCT = 265   // "struct"
const TK_BOOL = 266     // "_Bool"
const TK_IF = 267       // "if"
const TK_ELSE = 268     // "else"
const TK_FOR = 269      // "for"
const TK_DO = 270       // "do"
const TK_WHILE = 271    // "while"
const TK_SWITCH = 272   // "switch"
const TK_CASE = 273     // "case"
const TK_BREAK = 274    // "break"
const TK_CONTINUE = 275 // "continue"
const TK_EQ = 276       // ==
const TK_NE = 277       // !=
const TK_LE = 278       // <=
const TK_GE = 279       // >=
const TK_LOGOR = 280    // ||
const TK_LOGAND = 281   // &&
const TK_SHL = 282      // <<
const TK_SHR = 283      // >>
const TK_INC = 284      // ++
const TK_DEC = 285      // --
const TK_MUL_EQ = 286   // *=
const TK_DIV_EQ = 287   // /=
const TK_MOD_EQ = 288   // %=
const TK_ADD_EQ = 289   // +=
const TK_SUB_EQ = 290   // -=
const TK_SHL_EQ = 291   // <<=
const TK_SHR_EQ = 292   // >>=
const TK_AND_EQ = 293   // &=
const TK_XOR_EQ = 294   // ^=
const TK_OR_EQ = 295    // |=
const TK_RETURN = 296   // "return"
const TK_SIZEOF = 297   // "sizeof"
const TK_ALIGNOF = 298  // "_Alignof"
const TK_TYPEOF = 299   // "typeof"
const TK_PARAM = 300    // Function-like macro parameter
const TK_EOF = 301      // End marker

// Token type
type Token struct {
	ty   int    // Token type
	val  int    // Number literal
	name string // Identifier

	// String literal
	str string
	len int

	// For preprocessor
	stringize bool

	// For errorReport reporting
	ctx   *Context
	start int
	end   int
}

// parse.go
const (
	ND_NUM       = iota + 256 // Number literal
	ND_STR                    // String literal
	ND_IDENT                  // Identigier
	ND_STRUCT                 // Struct
	ND_DECL                   // declaration
	ND_VARDEF                 // Variable definition
	ND_LVAR                   // Local variable reference
	ND_GVAR                   // Global variable reference
	ND_IF                     // "if"
	ND_FOR                    // "for"
	ND_DO_WHILE               // do ... while
	ND_BREAK                  // break
	ND_ADDR                   // address-of operator ("&")
	ND_DEREF                  // pointer dereference ("*")
	ND_DOT                    // Struct member access
	ND_EQ                     // ==
	ND_NE                     // !=
	ND_LE                     // <=
	ND_LOGOR                  // ||
	ND_LOGAND                 // &&
	ND_SHL                    // <<
	ND_SHR                    // >>
	ND_MOD                    // %
	ND_NEG                    // -
	ND_POST_INC               // post ++
	ND_POST_DEC               // post --
	ND_MUL_EQ                 // *=
	ND_DIV_EQ                 // /=
	ND_MOD_EQ                 // %=
	ND_ADD_EQ                 // +=
	ND_SUB_EQ                 // -=
	ND_SHL_EQ                 // <<=
	ND_SHR_EQ                 // >>=
	ND_BITAND_EQ              // &=
	ND_XOR_EQ                 // ^=
	ND_BITOR_EQ               // |=
	ND_RETURN                 // "return"
	ND_SIZEOF                 // "sizeof"
	ND_ALIGNOF                // "_Alignof"
	ND_CALL                   // Function call
	ND_FUNC                   // Function definition
	ND_COMP_STMT              // Compound statement
	ND_EXPR_STMT              // Expressions statement
	ND_STMT_EXPR              // Statement expression (GUN extn.)
	ND_NULL                   // Null statement
)

const (
	INT = iota
	CHAR
	VOID
	PTR
	ARY
	STRUCT
	FUNC
)

type Node struct {
	op    int     // Node type
	ty    *Type   // C type
	lhs   *Node   // left-hand side
	rhs   *Node   // right-hand side
	val   int     // Number literal
	expr  *Node   // "return" or expression stmt
	stmts *Vector // Compound statement

	name string // Identifier

	// Global variable
	is_extern bool
	data      string
	len       int

	// "if" ( cond ) then "else" els
	// "for" ( init; cond; inc ) body
	cond *Node
	then *Node
	els  *Node
	init *Node
	body *Node
	inc  *Node

	// Function definition
	stacksize int
	globals   *Vector

	// Offset from BP or beginning of a struct
	offset int

	// Function call
	args *Vector
}

// sema.go

type Var struct {
	ty       *Type
	is_local bool

	// local
	offset int

	// global
	name      string
	is_extern bool
	data      string
	len       int
}

// ir_dump.go

type IRInfo struct {
	name string
	ty   int
}

// gen_ir.go

const (
	IR_ADD = iota + 256
	IR_SUB
	IR_MUL
	IR_DIV
	IR_IMM
	IR_BPREL
	IR_MOV
	IR_RETURN
	IR_CALL
	IR_LABEL
	IR_LABEL_ADDR
	IR_EQ
	IR_NE
	IR_LE
	IR_LT
	IR_AND
	IR_OR
	IR_XOR
	IR_SHL
	IR_SHR
	IR_MOD
	IR_NEG
	IR_JMP
	IR_IF
	IR_UNLESS
	IR_LOAD
	IR_STORE
	IR_STORE_ARG
	IR_KILL
	IR_NOP
)

type IR struct {
	op  int
	lhs int
	rhs int

	// Load/Store size in bytes
	size int

	// For binary operator. If true, rhs is an immediate.
	is_imm bool

	// Function call
	name  string
	nargs int
	args  [6]int
}

const (
	IR_TY_NOARG = iota + 256
	IR_TY_BINARY
	IR_TY_REG
	IR_TY_IMM
	IR_TY_MEM
	IR_TY_JMP
	IR_TY_LABEL
	IR_TY_LABEL_ADDR
	IR_TY_REG_REG
	IR_TY_REG_IMM
	IR_TY_STORE_ARG
	IR_TY_REG_LABEL
	IR_TY_CALL
)

type Function struct {
	name      string
	stacksize int
	globals   *Vector
	ir        *Vector
}
