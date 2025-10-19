package main

import (
	"reflect"
	"testing"
)

// 接口类型可以持有任意类型的值，但默认只能访问接口定义的方法。
// 通过类型断言，可以访问接口持有的具体类型的值和方法。

// go支持两种类型断言：
// 1. 单值断言, 如果断言失败会引发(panic)
// 2. 双值断言, 如果断言失败不会引发panic，而是返回一个布尔值

// 语法:
// v := x.(T) // 单值断言
// v, ok := x.(T) // 双值断言

// 如果接口值为 nil, 那么任何类型的断言都会失败。此时单值断言会引发恐慌(panic)，多值断言会返回断言类型的零值和 false。
func TestTypeAssert(t *testing.T) {
	var x interface{} = "hello"
	s := x.(string) // 单值断言, s 为 string 类型
	t.Log(s)

	s, ok := x.(string) // 双值断言
	if ok {
		t.Log(s)
	} else {
		t.Log("x 不是 string 类型")
	}
}

/*

switch v := i.(type) {
case T1:
    // v 是 T1 类型的值
case T2:
    // v 是 T2 类型的值
default:
    // 默认分支
}

*/

// 怎么判断某个类型是否实现了某个接口？
// 1. 编译时通过类型转换检查
type MyInterface interface {
	MyMethod()
}
type MyStruct struct{}

func (m MyStruct) MyMethod() {}

// 一般在包外部使用，将nil值赋给接口类型变量，编译器会检查该类型是否实现了接口(或者说将nil类型转化为MyStruct指针类型)
// nil是slice、map、指针、接口、函数、channel的零值
var _ MyInterface = (*MyStruct)(nil) // 编译时检查 MyStruct 是否实现了 MyInterface

// 2. 运行时使用断言检查
func TestInterfaceImplementation(t *testing.T) {
	var i interface{} = MyStruct{}
	if v, ok := i.(MyInterface); ok {
		t.Log("i 实现了 MyInterface")
		v.MyMethod() // 可以调用接口方法
	} else {
		t.Log("i 没有实现 MyInterface")
	}
}

// 3. 使用反射进行检查，但开销较高
func TestReflect(t *testing.T) {
	var i interface{} = MyStruct{}
	if reflect.TypeOf(i).Implements(reflect.TypeOf((*MyInterface)(nil)).Elem()) {
		t.Log("i 实现了 MyInterface")
	} else {
		t.Log("i 没有实现 MyInterface")
	}
	mapping := map[string]int{"a": 1, "b": 2}
	mapping["c"] = 3
}
