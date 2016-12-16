package rubykube

/*
  verbs.go is a collection of the verbs used to do stuff.
*/

import (
	"fmt"
	_ "io"
	_ "io/ioutil"
	_ "os"
	_ "path"
	_ "path/filepath"
	"strings"

	mruby "github.com/mitchellh/go-mruby"

	_ "k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api/v1"
)

// Definition is a jump table definition used for programming the DSL into the
// mruby interpreter.
type verbDefinition struct {
	verbFunc verbFunc
	argSpec  mruby.ArgSpec
}

// verbJumpTable is the dispatch instructions sent to the builder at preparation time.
var verbJumpTable = map[string]verbDefinition{
	//"debug":      {debug, mruby.ArgsOpt(1)},
	//"flatten":    {flatten, mruby.ArgsNone()},
	//"tag":        {tag, mruby.ArgsReq(1)},
	//"copy":       {doCopy, mruby.ArgsReq(2)},
	//"from":       {from, mruby.ArgsReq(1)},
	//"run":        {run, mruby.ArgsAny()},
	//"user":       {user, mruby.ArgsReq(1)},
	//"with_user":  {withUser, mruby.ArgsBlock() | mruby.ArgsReq(2)},
	//"workdir":    {workdir, mruby.ArgsReq(1)},
	//"inside":     {inside, mruby.ArgsBlock() | mruby.ArgsReq(2)},
	//"env":        {env, mruby.ArgsAny()},
	//"cmd":        {cmd, mruby.ArgsAny()},
	//"entrypoint": {entrypoint, mruby.ArgsAny()},
	//"set_exec":   {setExec, mruby.ArgsReq(1)},
	"new_app":    {newApp, mruby.ArgsReq(1)},
	"count_pods": {countPods, mruby.ArgsReq(0)},
	"pods":       {pods, mruby.ArgsReq(0)},
}

type verbFunc func(rk *RubyKube, args []*mruby.MrbValue, m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value)

func newApp(rk *RubyKube, args []*mruby.MrbValue, m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
	if err := standardCheck(rk, args, 1); err != nil {
		return nil, createException(m, err.Error())
	}

	container := kapi.Container{}

	err := iterateRubyHash(args[0], func(key, value *mruby.MrbValue) error {
		if value.Type() != mruby.TypeString {
			return fmt.Errorf("Value for key %q is not string, must be string", key.String())
		}

		//strArgs := []string{}
		//a := value.Array()

		//for i := 0; i < a.Len(); i++ {
		//	val, err := a.Get(i)
		//	if err != nil {
		//		return err
		//	}
		//	strArgs = append(strArgs, val.String())
		//}

		switch key.String() {
		case "image":
			container.Image = value.String()
			imageParts := strings.Split(strings.Split(container.Image, ":")[0], "/")
			container.Name = imageParts[len(imageParts)-1]
		case "name":
			container.Name = value.String()
		default:
			return fmt.Errorf("new_app only accepts :image and :name as keys")
		}
		return nil
	})

	if err != nil {
		return nil, createException(m, err.Error())
	}

	pod := kapi.Pod{
		Spec: kapi.PodSpec{
			Containers: []kapi.Container{container},
		},
	}

	fmt.Printf("%#v\n", pod)

	return nil, nil
}

func pods(rk *RubyKube, args []*mruby.MrbValue, m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
	var (
		pods *kapi.PodList
		v    *mruby.MrbValue
		err  error
	)
	podsClass := m.DefineClass("RubyKubePods", nil)

	podsInitializeArgs := mruby.ArgsReq(0)
	podsInitialize := func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		var err error
		pods, err = rk.clientset.Core().Pods("").List(kapi.ListOptions{})
		if err != nil {
			return nil, createException(m, err.Error())
		}
		return self, nil

	}
	podsClass.DefineMethod("fetch!", podsInitialize, podsInitializeArgs)

	podsInspectArgs := mruby.ArgsReq(0)
	podsInspect := func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		for n, pod := range pods.Items {
			fmt.Printf("%d: %s/%s\n", n, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		}
		return self, nil
	}
	podsClass.DefineMethod("inspect", podsInspect, podsInspectArgs)

	podsGetItemArgs := mruby.ArgsReq(1)
	podsGetItem := func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		if err := standardCheck(rk, args, 1); err != nil {
			return nil, createException(m, err.Error())
		}
		n := args[0]
		if n.Type() != mruby.TypeFixnum {
			return nil, createException(m, "Argument must be a integer")
		}
		pod := pods.Items[n.Fixnum()]
		fmt.Printf("%d: %s/%s\n", n.Fixnum(), pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		return self, nil
	}
	podsClass.DefineMethod("[]", podsGetItem, podsGetItemArgs)

	if v, err = podsClass.New(); err != nil {
		return nil, createException(m, err.Error())
	}

	if v, err = v.Call("fetch!"); err != nil {
		return nil, createException(m, err.Error())
	}

	return v, nil
}

func countPods(rk *RubyKube, args []*mruby.MrbValue, m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
	//if err := standardCheck(rk, args, 1); err != nil {
	//	return nil, createException(m, err.Error())
	//}

	pods, err := rk.clientset.Core().Pods("").List(kapi.ListOptions{})
	if err != nil {
		return nil, createException(m, err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	return nil, nil
}