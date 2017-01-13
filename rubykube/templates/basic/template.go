package basic

import (
	mruby "github.com/mitchellh/go-mruby"
)

// template type RubyKubeClass(classNameString, instanceVariableName, instanceVariableType)
type classNameString string
type instanceVariableName int
type instanceVariableType int

type RubyKubeClass struct {
	class   *mruby.Class
	objects []RubyKubeClassInstance
	rk      *RubyKube
}

type RubyKubeClassInstance struct {
	self *mruby.MrbValue
	vars *RubyKubeClassInstanceVars
}

type RubyKubeClassInstanceVars struct {
	instanceVariableName *instanceVariableType
}

func NewRubyKubeClass(rk *RubyKube) *RubyKubeClass {
	c := &RubyKubeClass{objects: []RubyKubeClassInstance{}, rk: rk}
	c.class = DefineRubyKubeClass(rk, c)
	return c
}

func DefineRubyKubeClass(rk *RubyKube, c *RubyKubeClass) *mruby.Class {
	// common methods
	return rk.defineClass(classNameString, map[string]methodDefintion{
		"object_count": {
			mruby.ArgsNone(), func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
				return m.FixnumValue(len(c.objects)), nil
			},
			classMethod,
		},
	})
}

func (c *RubyKubeClass) New() (*RubyKubeClassInstance, error) {
	s, err := c.class.New()
	if err != nil {
		return nil, err
	}
	o := RubyKubeClassInstance{
		self: s,
		vars: &RubyKubeClassInstanceVars{
			&instanceVariableType{},
		},
	}
	c.objects = append(c.objects, o)
	return &o, nil
}

func (c *RubyKubeClass) LookupVars(this *mruby.MrbValue) (*RubyKubeClassInstanceVars, error) {
	for _, that := range c.objects {
		if *this == *that.self {
			return that.vars, nil
		}
	}
	return nil, fmt.Errorf("%s: could not find class instance", classNameString)
}
