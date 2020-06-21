package go9cc

// C preprocessor

var (
	macros *Map
)

const (
	OBJLIKE = iota
	FUNCLIKE
)

type Context_p struct {
	input  *Vector
	output *Vector
	pos    int
	next   *Context_p
}

type Macro struct {
	ty     int
	tokens *Vector
	params *Vector
}

func new_ctx_p(next *Context_p, input *Vector) *Context_p {
	c := new(Context_p)
	c.input = input
	c.output = new_vec()
	c.next = next
	return c
}

func new_macro(ty int, name string) *Macro {
	m := new(Macro)
	m.ty = ty
	m.tokens = new_vec()
	m.params = new_vec()
	map_put(macros, name, m)
	return m
}

func (ctx_p *Context_p)append_p(v *Vector) {
	for i := 0; i < v.len; i++ {
		vec_push(ctx_p.output, v.data[i])
	}
}

func (ctx_p *Context_p)add_p(t *Token) { vec_push(ctx_p.output, t) }

func (ctx_p *Context_p)next_p() *Token {
	// assert(ctx_p,pos < ctx_p.input.len)
	t := ctx_p.input.data[ctx_p.pos].(*Token)
	ctx_p.pos++
	return t
}

func (ctx_p *Context_p)eof() bool { return ctx_p.pos == ctx_p.input.len }

func (ctx_p *Context_p)get(ty int, msg string) *Token {
	t := ctx_p.next_p()
	if t.ty != ty {
		bad_token(t, msg)
	}
	return t
}

func (ctx_p *Context_p)ident_p(msg string) string {
	t := ctx_p.get(TK_IDENT, "parameter file expected")
	return t.name
}

func (ctx_p *Context_p)peek() *Token { return ctx_p.input.data[ctx_p.pos].(*Token) }

func (ctx_p *Context_p)consume_p(ty int) bool {
	if ctx_p.peek().ty != ty {
		return false
	}
	ctx_p.pos++
	return true
}

func (ctx_p *Context_p)read_until_eol() *Vector {
	v := new_vec()
	for !ctx_p.eof() {
		t := ctx_p.next_p()
		if t.ty == '\n' {
			break
		}
		vec_push(v, t)
	}
	return v
}

func new_int_p(val int) *Token {
	t := new(Token)
	t.ty = TK_NUM
	t.val = val
	return t
}

func new_param(val int) *Token {
	t := new(Token)
	t.ty = TK_PARAM
	t.val = val
	return t
}

func is_ident(t *Token, s string) bool {
	return t.ty == TK_IDENT && strcmp(t.name, s) == 0
}

func replace_params(m *Macro) {
	params := m.params
	tokens := m.tokens

	// Replaces macro parameter tokens with TK_PARAM tokens
	mm := new_map()
	for i := 0; i < params.len; i++ {
		name := params.data[i].(string)
		map_puti(mm, name, i)
	}

	for i := 0; i < tokens.len; i++ {
		t := tokens.data[i].(*Token)
		if t.ty != TK_IDENT {
			continue
		}
		n := map_geti(mm, t.name, -1)
		if n == -1 {
			continue
		}
		tokens.data[i] = new_param(n)
	}

	// Process '#' followed by a macro parameter.
	v := new_vec()
	for i := 0; i < tokens.len; i++ {
		t1 := tokens.data[i].(*Token)
		t2 := tokens.data[i+1]

		if i != tokens.len-1 && t1.ty == '#' && t2.(*Token).ty == TK_PARAM {
			t2.(*Token).stringize = true
			vec_push(v, t2)
			i++
		} else {
			vec_push(v, t1)
		}
	}
	m.tokens = v
}

func (ctx_p *Context_p)read_one_arg() *Vector {
	v := new_vec()
	start := ctx_p.peek()
	level := 0

	for !ctx_p.eof() {
		t := ctx_p.peek()
		if level == 0 {
			if t.ty == ')' || t.ty == ',' {
				return v
			}
		}

		ctx_p.next_p()
		if t.ty == '(' {
			level++
		} else if t.ty == ')' {
			level--
		}
		vec_push(v, t)
	}
	bad_token(start, "unclosed macro argument")
	return nil
}

func (ctx_p *Context_p)read_args() *Vector {
	v := new_vec()
	if ctx_p.consume_p(')') {
		return v
	}
	vec_push(v, ctx_p.read_one_arg())
	for !ctx_p.consume_p(')') {
		ctx_p.get(',', "comma expected")
		vec_push(v, ctx_p.read_one_arg())
	}
	return v
}

func stringize(tokens *Vector) *Token {
	sb := new_sb()

	for i := 0; i < tokens.len; i++ {
		t := tokens.data[i].(*Token)
		if i != 0 {
			sb_add(sb, " ")
		}
		sb_append(sb, tokstr(t))
	}

	t := new(Token)
	t.ty = TK_STR
	t.str = sb_get(sb)
	t.len = sb.len
	return t
}

func (ctx_p *Context_p)apply(m *Macro, start *Token) {
	if m.ty == OBJLIKE {
		ctx_p.append_p(m.tokens)
		return
	}

	// Function-like macro
	ctx_p.get('(', "comma expected")
	args := ctx_p.read_args()
	if m.params.len != args.len {
		bad_token(start, "number of parameter does not match")
	}

	for i := 0; i < m.tokens.len; i++ {
		t := m.tokens.data[i].(*Token)

		if is_ident(t, "__LINE__") {
			ctx_p.add_p(new_int_p(line(t)))
			continue
		}

		if t.ty == TK_PARAM {
			if t.stringize {
				ctx_p.add_p(stringize(args.data[t.val].(*Vector)))
			} else {
				ctx_p.append_p(args.data[t.val].(*Vector))
			}
			continue
		}
		ctx_p.add_p(t)
	}
}

func (ctx_p *Context_p)funclike_macro(name string) {
	m := new_macro(FUNCLIKE, name)
	vec_push(m.params, ctx_p.ident_p("parameter file expected"))
	for !ctx_p.consume_p(')') {
		ctx_p.get(',', "comma expected")
		vec_push(m.params, ctx_p.ident_p("parameter file expected"))
	}
	m.tokens = ctx_p.read_until_eol()
	replace_params(m)
}

func (ctx_p *Context_p)objlike_macro(name string) {
	m := new_macro(OBJLIKE, name)
	m.tokens = ctx_p.read_until_eol()
}

func (ctx_p *Context_p)define() {
	name := ctx_p.ident_p("macro file expected")
	if ctx_p.consume_p('(') {
		ctx_p.funclike_macro(name)
		return
	}
	ctx_p.objlike_macro(name)
}

func (app *TokenApp) include() {
	ctx_p := app.ctx_p

	t := ctx_p.get(TK_STR, "string expected")
	path := t.str
	ctx_p.get('\n', "newline expected")
	ctx_p.append_p(Tokenize(path, false, app.ctx))
}

func (app *TokenApp) preprocess(tokens *Vector) *Vector {
	if macros == nil {
		macros = new_map()
	}
	app.ctx_p = new_ctx_p(app.ctx_p, tokens)

	ctx_p := app.ctx_p
	for !ctx_p.eof() {
		t := ctx_p.next_p()

		if t.ty == TK_IDENT {
			m := map_get(macros, t.name)
			if m != nil {
				ctx_p.apply(m.(*Macro), t)
			} else {
				ctx_p.add_p(t)
			}
			continue
		}

		if t.ty != '#' {
			ctx_p.add_p(t)
			continue
		}

		t = ctx_p.get(TK_IDENT, "identifier expected")

		if strcmp(t.name, "define") == 0 {
			ctx_p.define()
		} else if strcmp(t.name, "include") == 0 {
			app.include()
		} else {
			bad_token(t, "unknown directive")
		}
	}

	v := app.ctx_p.output
	app.ctx_p = app.ctx_p.next
	return v
}
