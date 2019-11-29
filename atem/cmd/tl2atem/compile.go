package main

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/metaleap/atmo/atem"
	tl "github.com/metaleap/go-machines/toylam"
)

var defsDone = map[string]int{}

func compile(mainTopDefQName string) {
	compileTopDef(tl.StdRequiredDefs_id)
	compileTopDef(tl.StdRequiredDefs_true)
	compileTopDef(tl.StdRequiredDefs_false)
	compileTopDef(tl.StdRequiredDefs_listNil)
	compileTopDef(tl.StdRequiredDefs_listCons)

	idx, meta := compileTopDef(mainTopDefQName), []string{mainTopDefQName}
	outProg = append(outProg, outProg[idx])
	outProg[idx] = FuncDef{Meta: meta, Body: ExprFuncRef(len(outProg) - 1)}

	for again := false; again; {
		outProg, again = optimize(outProg)
	}
}

func compileExpr(expr tl.Expr, curFunc *FuncDef, curFuncsArgs []*tl.ExprFunc) Expr {
	switch it := expr.(type) {
	case *tl.ExprLitNum:
		return ExprNumInt(it.NumVal)
	case *tl.ExprCall:
		return ExprAppl{Callee: compileExpr(it.Callee, curFunc, curFuncsArgs), Arg: compileExpr(it.CallArg, curFunc, curFuncsArgs)}
	case *tl.ExprName:
		if it.IdxOrInstr < 0 {
			argidx := len(curFuncsArgs) + it.IdxOrInstr
			return -ExprArgRef(1 + argidx)
		} else if it.IdxOrInstr > 0 {
			if opcode, ok := instr2op[tl.Instr(it.IdxOrInstr)]; ok {
				return ExprFuncRef(opcode)
			}
		} else {
			for i, farg := range curFuncsArgs {
				if farg.ArgName == it.NameVal {
					return -ExprArgRef(1 + i)
				}
			}
			idx := compileTopDef(it.NameVal)
			return ExprFuncRef(idx)
		}
	case *tl.ExprFunc:
		globalname, globalbody := "//"+curFunc.Meta[0]+"/lam/"+it.ArgName+strconv.Itoa(len(inProg.TopDefs)), it
		freevars, expr := map[string]int{}, tl.Expr(&tl.ExprName{NameVal: globalname})
		freeVars(globalbody, map[string]string{}, freevars)
		for fvname := range freevars {
			globalbody = &tl.ExprFunc{ArgName: fvname, Body: globalbody}
		}
		fargs, _ := dissectFunc(globalbody)
		for _, farg := range fargs[:len(freevars)] {
			expr = &tl.ExprCall{Callee: expr, CallArg: &tl.ExprName{NameVal: farg.ArgName}}
		}
		inProg.TopDefs[globalname] = globalbody
		return compileExpr(expr, curFunc, curFuncsArgs)
	}
	panic(fmt.Sprintf("%T\t%v", expr, expr))
}

func compileTopDef(name string) int {
	idx, done := defsDone[name]
	if !done {
		prefstd, topdef, locals := "", inProg.TopDefs[name], inProg.TopDefSepLocals[name]
		if topdef == nil {
			if prefstd, topdef = tl.StdModuleName+".", inProg.TopDefs[tl.StdModuleName+"."+name]; topdef == nil {
				for k, v := range inProg.TopDefs {
					if strings.HasSuffix(k, "."+name) && strings.HasPrefix(k, tl.StdModuleName+".") {
						prefstd, topdef = k[:len(k)-len(name)], v
						break
					}
				}
			}
			if idx, done = defsDone[prefstd+name]; done {
				return idx
			}
			locals = inProg.TopDefSepLocals[prefstd+name]
		}
		if topdef == nil {
			panic(name)
		}
		idx, name = len(outProg), prefstd+name
		outProg, defsDone[name] = append(outProg, FuncDef{Meta: []string{"[" + strconv.Itoa(idx) + "]" + name}}), idx

		result := &outProg[idx]
		curfuncsargs, body := dissectFunc(topdef)
		result.Args = make([]int, len(curfuncsargs))
		for i, f := range curfuncsargs {
			result.Meta = append(result.Meta, f.ArgName)
			result.Args[i] = body.ReplaceName(f.ArgName, f.ArgName) // just counts occurrences
			for _, local := range locals {
				if numrefs := local.Expr.ReplaceName(f.ArgName, f.ArgName); numrefs > 0 {
					result.Args[i] += numrefs
				}
			}
		}

		localgnames := map[string]string{}
		var lifted func(int, []*tl.ExprFunc)
		lifted = func(idx int, argsadded []*tl.ExprFunc) {
			gname := localgnames[locals[idx].Name]
			var expr tl.Expr = &tl.ExprName{NameVal: gname}
			for _, farg := range argsadded {
				expr = &tl.ExprCall{Callee: expr, CallArg: &tl.ExprName{NameVal: farg.ArgName}}
			}
			body = body.RewriteName(gname, expr)
			for j, exprcopy := 0, fullCopy(expr); j <= idx; j++ {
				if jname := localgnames[locals[j].Name]; 0 < inProg.TopDefs[jname].ReplaceName(gname, gname) {
					nuargs := make([]*tl.ExprFunc, 0, len(argsadded))
					ownargs, _ := dissectFunc(inProg.TopDefs[jname])
					for _, farg := range argsadded {
						found := false
						for _, ownarg := range ownargs {
							if found = (ownarg.ArgName == farg.ArgName); found {
								break
							}
						}
						if !found {
							nuargs = append(nuargs, farg)
						}
					}
					exprcopy, inProg.TopDefs[jname] = fullCopy(expr), inProg.TopDefs[jname].RewriteName(gname, exprcopy)
					if len(nuargs) > 0 {
						for i := len(nuargs) - 1; i >= 0; i-- {
							inProg.TopDefs[jname] = &tl.ExprFunc{ArgName: nuargs[i].ArgName, Body: inProg.TopDefs[jname]}
						}
						lifted(j, nuargs)
					}
				}
			}
		}

		for i, local := range locals {
			gname := "//" + name + "/local/" + local.Name + strconv.Itoa(i)
			localgnames[local.Name], body = gname, body.RewriteName(local.Name, &tl.ExprName{NameVal: gname})
			for j := 0; j <= i; j++ {
				locals[j].Expr = locals[j].Expr.RewriteName(local.Name, &tl.ExprName{NameVal: gname})
			}
		}
		for i, local := range locals {
			globalname, globalbody := localgnames[local.Name], local.Expr
			freevars := map[string]int{}
			freeVars(local.Expr, localgnames, freevars)
			for fvname := range freevars {
				globalbody = &tl.ExprFunc{ArgName: fvname, Body: globalbody}
			}
			inProg.TopDefs[globalname] = globalbody
			fargs, _ := dissectFunc(globalbody)
			lifted(i, fargs[:len(freevars)])
		}

		result.Body = compileExpr(body, result, curfuncsargs)
		outProg[idx] = *result // crucial as the slice could have been resized by now (even tho we give it an initial cap that makes this unlikely, this safeguard will work for any outrageous program size)
	}
	return idx
}
