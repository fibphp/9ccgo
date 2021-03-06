package go9cc

// This is recursice-descendent parser which constructs abstract
// syntax tree from input tokens.
//
// This parser knows only about BNF of the C grammer and doesn't care
// about its semantics. Therefore, some invalid expressions, such as
// `1+2=3`, are accepted by this parser, but that's intentional.
// Semantic errors are detected in a later pass.

var (
	pos        = 0
	penv       *PEnv
	tokens     *Vector
	int_ty     = Type{ty: INT, size: 4, align: 4}
	null_stmt  = Node{op: ND_NULL}
	break_stmt = Node{op: ND_BREAK}
)

type PEnv struct {
	typedefs *Map
	tags     *Map
	next     *PEnv
}

func new_penv(next *PEnv) *PEnv {
	env := new(PEnv)
	env.typedefs = new_map()
	env.tags = new_map()
	env.next = next
	return env
}

func find_typedef(name string) *Type {
	for e := penv; e != nil; e = e.next {
		ty := map_get(e.typedefs, name)
		if ty != nil {
			return ty.(*Type)
		}
	}
	return nil
}

func find_tag(name string) *Type {
	for e := penv; e != nil; e = e.next {
		ty := map_get(e.tags, name)
		if ty != nil {
			return ty.(*Type)
		}
	}
	return nil
}

func expect(ty int) {
	t := tokens.data[pos].(*Token)
	if t.ty == ty {
		pos++
		return
	}

	if isprint(uint8(ty)) {
		bad_token(t, format("%c expected", ty))
	}
	// assert(ty == TK_WHILE)
	//bad_token(t, format("'while' expected", ty))
	bad_token(t, "'while' expected")
}

func new_prim_ty(ty, size int) *Type {
	ret := new(Type)
	ret.ty = ty
	ret.size = size
	ret.align = size
	return ret
}

func void_tyf() *Type { return new_prim_ty(VOID, 0) }
func char_tyf() *Type { return new_prim_ty(CHAR, 1) }
func int_tyf() *Type  { return new_prim_ty(INT, 4) }

func consume(ty int) bool {
	t := tokens.data[pos].(*Token)
	if t.ty != ty {
		return false
	}
	pos++
	return true
}

func is_typename() bool {
	t := tokens.data[pos].(*Token)
	if t.ty == TK_IDENT {
		ret := find_typedef(t.name)
		return ret != nil
	}
	return t.ty == TK_INT || t.ty == TK_CHAR || t.ty == TK_VOID || t.ty == TK_STRUCT
}

func add_members(ty *Type, members *Vector) {
	off := 0
	for i := 0; i < members.len; i++ {
		node := members.data[i].(*Node)
		//assert(node.op == ND_VARDEF)

		t := node.ty
		off = roundup(off, t.align)
		t.offset = off
		off += t.size

		if ty.align < node.ty.align {
			ty.align = node.ty.align
		}
	}

	ty.members = members
	ty.size = roundup(off, ty.align)
}

func decl_specifiers() *Type {
	t := tokens.data[pos].(*Token)
	pos++

	if t.ty == TK_IDENT {
		ty := find_typedef(t.name)
		if ty == nil {
			pos--
		}
		return ty
	}

	if t.ty == TK_INT {
		return int_tyf()
	}

	if t.ty == TK_CHAR {
		return char_tyf()
	}

	if t.ty == TK_VOID {
		return void_tyf()
	}

	if t.ty == TK_STRUCT {
		var tag string
		t := tokens.data[pos].(*Token)
		if t.ty == TK_IDENT {
			pos++
			tag = t.name
		}

		var members *Vector
		if consume('{') {
			members = new_vec()
			for !consume('}') {
				vec_push(members, declaration())
			}
		}

		if tag == "" && members == nil {
			bad_token(t, "bad struct definition")
		}

		var ty *Type
		if tag != "" && members == nil {
			ty = find_tag(tag)
		}

		if ty == nil {
			ty = new(Type)
			ty.ty = STRUCT
		}

		if members != nil {
			add_members(ty, members)
			if tag != "" {
				map_put(penv.tags, tag, ty)
			}
		}
		return ty
	}

	bad_token(t, "typename expected")
	return nil
}

func new_binop(op int, lhs, rhs *Node) *Node {
	node := new(Node)
	node.op = op
	node.lhs = lhs
	node.rhs = rhs
	return node
}

func new_expr(op int, expr *Node) *Node {
	node := new(Node)
	node.op = op
	node.expr = expr
	return node
}

func new_num(val int) *Node {
	node := new(Node)
	node.op = ND_NUM
	node.ty = int_tyf()
	node.val = val
	return node
}

func ident() string {
	t := tokens.data[pos].(*Token)
	pos++
	if t.ty != TK_IDENT {
		bad_token(t, "identifier expected")
	}
	return t.name
}

func primary() *Node {
	t := tokens.data[pos].(*Token)
	pos++

	if t.ty == '(' {
		if consume('{') {
			node := new(Node)
			node.op = ND_STMT_EXPR
			node.body = compound_stmt()
			expect(')')
			return node
		}
		node := expr()
		expect(')')
		return node
	}

	node := new(Node)
	if t.ty == TK_NUM {
		return new_num(t.val)
	}

	if t.ty == TK_STR {
		node.ty = ary_of(char_tyf(), t.len+1) // +1 is '\0'
		node.op = ND_STR
		node.data = t.str
		node.len = t.len
		return node
	}

	if t.ty == TK_IDENT {
		node.name = t.name

		if !consume('(') {
			node.op = ND_IDENT
			return node
		}

		node.op = ND_CALL
		node.args = new_vec()
		if consume(')') {
			return node
		}

		vec_push(node.args, assign())
		for consume(',') {
			vec_push(node.args, assign())
		}
		expect(')')
		return node
	}

	bad_token(t, "primary expression expected")
	return nil
}

func postfix() *Node {
	lhs := primary()

	for {
		if consume(TK_INC) {
			lhs = new_expr(ND_POST_INC, lhs)
			continue
		}

		if consume(TK_DEC) {
			lhs = new_expr(ND_POST_DEC, lhs)
			continue
		}

		if consume('.') {
			lhs = new_expr(ND_DOT, lhs)
			lhs.name = ident()
			continue
		}

		if consume(TK_ARROW) {
			lhs = new_expr(ND_DOT, new_expr(ND_DEREF, lhs))
			lhs.name = ident()
			continue
		}

		if consume('[') {
			lhs = new_expr(ND_DEREF, new_binop('+', lhs, assign()))
			expect(']')
			continue
		}
		return lhs
	}
	return nil
}

func unary() *Node {
	if consume('-') {
		return new_expr(ND_NEG, unary())
	}
	if consume('*') {
		return new_expr(ND_DEREF, unary())
	}
	if consume('&') {
		return new_expr(ND_ADDR, unary())
	}
	if consume('!') {
		return new_expr('!', unary())
	}
	if consume('~') {
		return new_expr('~', unary())
	}
	if consume(TK_SIZEOF) {
		return new_expr(ND_SIZEOF, unary())
	}
	if consume(TK_ALIGNOF) {
		return new_expr(ND_ALIGNOF, unary())
	}

	if consume(TK_INC) {
		return new_binop(ND_ADD_EQ, unary(), new_num(1))
	}
	if consume(TK_DEC) {
		return new_binop(ND_SUB_EQ, unary(), new_num(1))
	}

	return postfix()
}

func mul() *Node {
	lhs := unary()
	for {
		if consume('*') {
			lhs = new_binop('*', lhs, unary())
		} else if consume('/') {
			lhs = new_binop('/', lhs, unary())
		} else if consume('%') {
			lhs = new_binop('%', lhs, unary())
		} else {
			return lhs
		}
	}
	return lhs
}

func read_array(ty *Type) *Type {
	v := new_vec()

	for consume('[') {
		if consume(']') {
			vec_push(v, -1)
			continue
		}

		t := tokens.data[pos].(*Token)
		l := expr()
		if l.op != ND_NUM {
			bad_token(t, "number expected")
		}
		vec_push(v, l.val)
		expect(']')
	}
	for i := v.len - 1; i >= 0; i-- {
		l := v.data[i].(int)
		ty = ary_of(ty, l)
	}
	return ty
}

func parse_add() *Node {
	lhs := mul()
	for {
		if consume('+') {
			lhs = new_binop('+', lhs, mul())
		} else if consume('-') {
			lhs = new_binop('-', lhs, mul())
		} else {
			return lhs
		}
	}
	return lhs
}

func shift() *Node {
	lhs := parse_add()
	for {
		if consume(TK_SHL) {
			lhs = new_binop(ND_SHL, lhs, parse_add())
		} else if consume(TK_SHR) {
			lhs = new_binop(ND_SHR, lhs, parse_add())
		} else {
			return lhs
		}
	}
	return lhs
}

func relational() *Node {
	lhs := shift()
	for {
		if consume('<') {
			lhs = new_binop('<', lhs, shift())
		} else if consume('>') {
			lhs = new_binop('<', shift(), lhs)
		} else if consume(TK_LE) {
			lhs = new_binop(ND_LE, lhs, shift())
		} else if consume(TK_GE) {
			lhs = new_binop(ND_LE, shift(), lhs)
		} else {
			return lhs
		}
	}
}

func equality() *Node {
	lhs := relational()
	for {
		if consume(TK_EQ) {
			lhs = new_binop(ND_EQ, lhs, relational())
		} else if consume(TK_NE) {
			lhs = new_binop(ND_NE, lhs, relational())
		} else {
			return lhs
		}
	}
}

func bit_and() *Node {
	lhs := equality()
	for consume('&') {
		lhs = new_binop('&', lhs, equality())
	}
	return lhs
}

func bit_xor() *Node {
	lhs := bit_and()
	for consume('^') {
		lhs = new_binop('^', lhs, bit_and())
	}
	return lhs
}

func bit_or() *Node {
	lhs := bit_xor()
	for consume('|') {
		lhs = new_binop('|', lhs, bit_xor())
	}
	return lhs
}

func logand() *Node {
	lhs := bit_or()
	for consume(TK_LOGAND) {
		lhs = new_binop(ND_LOGAND, lhs, bit_or())
	}
	return lhs
}

func logor() *Node {
	lhs := logand()
	for consume(TK_LOGOR) {
		lhs = new_binop(ND_LOGOR, lhs, logand())
	}
	return lhs
}

func conditional() *Node {
	cond := logor()
	if !consume('?') {
		return cond
	}

	node := new(Node)
	node.op = '?'
	node.cond = cond
	node.then = expr()
	expect(':')
	node.els = conditional()
	return node
}

func assignment_op() int {
	if consume('=') {
		return '='
	}
	if consume(TK_MUL_EQ) {
		return ND_MUL_EQ
	}
	if consume(TK_DIV_EQ) {
		return ND_DIV_EQ
	}
	if consume(TK_MOD_EQ) {
		return ND_MOD_EQ
	}
	if consume(TK_ADD_EQ) {
		return ND_ADD_EQ
	}
	if consume(TK_SUB_EQ) {
		return ND_SUB_EQ
	}
	if consume(TK_SHL_EQ) {
		return ND_SHL_EQ
	}
	if consume(TK_SHR_EQ) {
		return ND_SHR_EQ
	}
	if consume(TK_AND_EQ) {
		return ND_BITAND_EQ
	}
	if consume(TK_XOR_EQ) {
		return ND_XOR_EQ
	}
	if consume(TK_OR_EQ) {
		return ND_BITOR_EQ
	}
	return 0
}

func assign() *Node {
	lhs := conditional()
	op := assignment_op()
	if op != 0 {
		return new_binop(op, lhs, assign())
	}
	return lhs
}

func expr() *Node {
	lhs := assign()
	if !consume(',') {
		return lhs
	}
	return new_binop(',', lhs, expr())
}

func direct_decl(ty *Type) *Node {
	t := tokens.data[pos].(*Token)
	var node *Node
	placeholder := new(Type)

	if t.ty == TK_IDENT {
		node = new(Node)
		node.op = ND_VARDEF
		node.ty = placeholder
		node.name = ident()
	} else if consume('(') {
		node = declarator(placeholder)
		expect(')')
	} else {
		bad_token(t, "bad direct-declarator")
	}

	// Read the second half of type file (e.g. `[3][5]`).
	*placeholder = *read_array(ty)

	// Read an initializer.
	if consume('=') {
		node.init = assign()
	}
	return node
}

func declarator(ty *Type) *Node {
	for consume('*') {
		ty = ptr_to(ty)
	}
	return direct_decl(ty)
}

func declaration() *Node {
	ty := decl_specifiers()
	node := declarator(ty)
	expect(';')
	return node
}

func param_declaration() *Node {
	ty := decl_specifiers()
	node := declarator(ty)
	if node.ty.ty == ARY {
		node.ty = ptr_to(node.ty.ary_of)
	}
	return node
}

func expr_stmt() *Node {
	node := new_expr(ND_EXPR_STMT, expr())
	expect(';')
	return node
}

func stmt() *Node {
	node := new(Node)
	t := tokens.data[pos].(*Token)
	pos++

	switch t.ty {
	case TK_TYPEDEF:
		node := declaration()
		// assert(node.file)
		map_put(penv.typedefs, node.name, node.ty)
		return &null_stmt
	case TK_IF:
		node.op = ND_IF
		expect('(')
		node.cond = expr()
		expect(')')

		node.then = stmt()

		if consume(TK_ELSE) {
			node.els = stmt()
		}
		return node
	case TK_FOR:
		node.op = ND_FOR
		expect('(')

		if is_typename() {
			node.init = declaration()
		} else if consume(';') {
			node.init = &null_stmt
		} else {
			node.init = expr_stmt()
		}

		if !consume(';') {
			node.cond = expr()
			expect(';')
		}

		if !consume(')') {
			node.inc = new_expr(ND_EXPR_STMT, expr())
			expect(')')
		}

		node.body = stmt()
		return node
	case TK_WHILE:
		node.op = ND_FOR
		node.init = &null_stmt
		node.inc = &null_stmt
		expect('(')
		node.cond = expr()
		expect(')')
		node.body = stmt()
		return node
	case TK_DO:
		node.op = ND_DO_WHILE
		node.body = stmt()
		expect(TK_WHILE)
		expect('(')
		node.cond = expr()
		expect(')')
		expect(';')
		return node
	case TK_BREAK:
		return &break_stmt
	case TK_RETURN:
		node.op = ND_RETURN
		node.expr = expr()
		expect(';')
		return node
	case '{':
		node.op = ND_COMP_STMT
		node.stmts = new_vec()
		for !consume('}') {
			vec_push(node.stmts, stmt())
		}
		return node
	case ';':
		return &null_stmt
	default:
		pos--
		if is_typename() {
			return declaration()
		}
		return expr_stmt()
	}
	return nil
}

func compound_stmt() *Node {

	node := new(Node)
	node.op = ND_COMP_STMT
	node.stmts = new_vec()

	penv = new_penv(penv)
	for !consume('}') {
		vec_push(node.stmts, stmt())
	}
	penv = penv.next
	return node
}

func toplevel() *Node {
	is_typedef := consume(TK_TYPEDEF)
	is_extern := consume(TK_EXTERN)

	ty := decl_specifiers()
	for consume('*') {
		ty = ptr_to(ty)
	}

	name := ident()

	// Function
	if consume('(') {
		node := new(Node)
		node.name = name
		node.args = new_vec()

		node.ty = new(Type)
		node.ty.ty = FUNC
		node.ty.returning = ty

		if !consume(')') {
			vec_push(node.args, param_declaration())
			for consume(',') {
				vec_push(node.args, param_declaration())
			}
			expect(')')
		}

		if consume(';') {
			node.op = ND_DECL
			return node
		}

		node.op = ND_FUNC
		t := tokens.data[pos].(*Token)
		expect('{')
		if is_typedef {
			bad_token(t, "typedef has function definition")
		}
		node.body = compound_stmt()
		return node
	}

	ty = read_array(ty)
	expect(';')

	if is_typedef {
		map_put(penv.typedefs, name, ty)
		return nil
	}

	// Global variable
	node := new(Node)
	node.op = ND_VARDEF
	node.ty = ty
	node.name = name
	node.is_extern = is_extern

	if !is_extern {
		node.data = ""
		node.len = node.ty.size
	}
	return node
}

func Parse(tokens_ *Vector) *Vector {
	tokens = tokens_
	pos = 0
	penv = new_penv(penv)

	v := new_vec()
	for {
		t := tokens.data[pos].(*Token)
		if t.ty == TK_EOF {
			return v
		}
		node := toplevel()
		if node != nil {
			vec_push(v, node)
		}
	}
}
