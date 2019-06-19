package main

var (
	vars      *Map
	stacksize int
)

type Var struct {
	ty     *Type
	offset int
}

func swap(p, q **Node) {
	r := *p
	*p = *q
	*q = r
}

func addr_of(base *Node, ty *Type) *Node {
	node := new(Node)
	node.op = ND_ADDR
	node.ty = ptr_of(ty)
	node.expr = base
	return node
}

func walk(node *Node, decay bool) *Node {
	switch node.op {
	case ND_NUM:
		return node
	case ND_IDENT:
		{
			v := map_get(vars, node.name).(*Var)
			if v == nil {
				error("undetined variable: %s", node.name)
			}
			node.op = ND_LVAR
			node.offset = v.offset

			if decay && v.ty.ty == ARY {
				return addr_of(node, v.ty.ary_of)
			}
			node.ty = v.ty
			return node
		}
	case ND_VARDEF:
		{
			stacksize += size_of(node.ty)
			node.offset = stacksize
			v := new(Var)
			v.ty = node.ty
			v.offset = stacksize
			map_put(vars, node.name, v)

			if node.init != nil {
				node.init = walk(node.init, true)
			}
			return node
		}
	case ND_IF:
		node.cond = walk(node.cond, true)
		node.then = walk(node.then, true)
		if node.els != nil {
			node.els = walk(node.els, true)
		}
		return node
	case ND_FOR:
		node.init = walk(node.init, true)
		node.cond = walk(node.cond, true)
		node.inc = walk(node.inc, true)
		node.body = walk(node.body, true)
		return node
	case '+', '-':
		node.lhs = walk(node.lhs, true)
		node.rhs = walk(node.rhs, true)

		if node.rhs.ty.ty == PTR {
			swap(&node.lhs, &node.rhs)
		}
		if node.rhs.ty.ty == PTR {
			error("pointer %c pointer' is not defined", node.op)
		}

		node.ty = node.lhs.ty
		return node
	case '=':
		node.lhs = walk(node.lhs, false)
		node.rhs = walk(node.rhs, true)
		node.ty = node.lhs.ty
		return node
	case '*', '/', '<', ND_LOGAND, ND_LOGOR:
		node.lhs = walk(node.lhs, true)
		node.rhs = walk(node.rhs, true)
		node.ty = node.lhs.ty
		return node
	case ND_ADDR:
		node.expr = walk(node.expr, true)
		node.ty = ptr_of(node.expr.ty)
		return node
	case ND_DEREF:
		node.expr = walk(node.expr, true)
		if node.expr.ty.ty != PTR {
			error("operand must be a pointer")
		}
		node.ty = node.expr.ty.ptr_of
		return node
	case ND_RETURN:
		node.expr = walk(node.expr, true)
		return node
	case ND_CALL:
		for i := 0; i < node.args.len; i++ {
			node.args.data[i] = walk(node.args.data[i].(*Node), true)
		}
		node.ty = &int_ty
		return node
	case ND_FUNC:
		for i := 0; i < node.args.len; i++ {
			node.args.data[i] = walk(node.args.data[i].(*Node), true)
		}
		node.body = walk(node.body, true)
		return node
	case ND_COMP_STMT:
		for i := 0; i < node.stmts.len; i++ {
			node.stmts.data[i] = walk(node.stmts.data[i].(*Node), true)
		}
		return node
	case ND_EXPR_STMT:
		node.expr = walk(node.expr, true)
		return node
	default:
		//assert(0 && "unknouwn node type")
	}
	return nil
}

func sema(nodes *Vector) {
	for i := 0; i < nodes.len; i++ {
		node := nodes.data[i].(*Node)
		//assert(node.op == ND_FUNC)

		vars = new_map()
		stacksize = 0
		walk(node, true)
		node.stacksize = stacksize
	}
}
