package rubykube

import (
	"flag"
	_ "fmt"
	"strings"

	"github.com/erikh/box/builder/signal"
	"github.com/erikh/box/log"
	mruby "github.com/mitchellh/go-mruby"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")

type RubyKube struct {
	mrb       *mruby.Mrb
	clientset *kubernetes.Clientset
}

func keep(omitFuncs []string, name string) bool {
	for _, fun := range omitFuncs {
		if name == fun {
			return false
		}
	}
	return true
}

// NewRubyKube may return an error on mruby or k8s.io/client-go issues.
func NewRubyKube(omitFuncs []string) (*RubyKube, error) {
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	rk := &RubyKube{mrb: mruby.NewMrb()}

	rk.mrb.DisableGC()

	for name, def := range verbJumpTable {
		if keep(omitFuncs, name) {
			rk.AddVerb(name, def.verbFunc, def.argSpec)
		}
	}

	for name, def := range funcJumpTable {
		if keep(omitFuncs, name) {
			inner := def.fun
			fn := func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
				return inner(rk, m, self)
			}

			rk.mrb.TopSelf().SingletonClass().DefineMethod(name, fn, def.argSpec)
		}
	}

	signal.SetSignal(nil)

	rk.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return rk, nil
}

// AddVerb adds a function to the mruby dispatch as well as adding hooks around
// the call to ensure containers are committed and intermediate layers are
// cleared.
func (rk *RubyKube) AddVerb(name string, fn verbFunc, args mruby.ArgSpec) {
	hookFunc := func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		strArgs := extractStringArgs(args)

		log.BuildStep(name, strings.Join(strArgs, ", "))

		return fn(rk, args, m, self)
	}

	rk.mrb.TopSelf().SingletonClass().DefineMethod(name, hookFunc, args)
}

// Run the script.
func (rk *RubyKube) Run(script string) (*mruby.MrbValue, error) {
	if _, err := rk.mrb.LoadString(script); err != nil {
		return nil, err
	}

	return mruby.String("").MrbValue(rk.mrb), nil
}

// Close tears down all functions of the RubyKube, preparing it for exit.
func (rk *RubyKube) Close() error {
	rk.mrb.EnableGC()
	rk.mrb.FullGC()
	rk.mrb.Close()
	return nil
}