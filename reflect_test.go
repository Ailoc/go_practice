package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestReflectType(t *testing.T) {
	var x float64 = 3.14
	x_type := reflect.TypeOf(x)
	x_value := reflect.ValueOf(x)

	t.Log("Type:", x_type)                 // Type: float64
	t.Log("Kind:", x_type.Kind())          // Kind: float64
	t.Log("Value:", x_value)               // Value: 3.14
	t.Log("Value.Kind():", x_value.Kind()) // Value.Kind(): float64

	// 获取值，类型转换
	t.Log("Float value:", x_value.Float())         // Float value: 3.14
	t.Log("Interface value:", x_value.Interface()) // Interface value: 3.14

	// 通过反射修改值
	p := reflect.ValueOf(&x) // 获取 x 的指针
	p.Elem().SetFloat(6.28)  // 修改 x 的值
	// CanSet() 检查可修改性（必须是导出字段且可寻址）
	t.Log("Modified x:", x) // Modified x: 6.28
}

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStruct(tt *testing.T) {
	p := Person{Name: "Alice", Age: 30}
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)

	// 遍历字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		tag := field.Tag.Get("json") // 获取标签

		fmt.Printf("字段: %s, 类型: %s, 值: %v, JSON Tag: %s\n",
			field.Name, field.Type, value.Interface(), tag)
	}
	// 输出:
	// 字段: Name, 类型: string, 值: Alice, JSON Tag: name
	// 字段: Age, 类型: int, 值: 30, JSON Tag: age

	// 修改字段（需指针）
	vp := reflect.ValueOf(&p).Elem()
	vp.FieldByName("Age").SetInt(31)
	fmt.Println("修改后 Age:", p.Age) // 输出: 修改后 Age: 31
}

func TestSlice_Map(tt *testing.T) {
	// 切片
	slice := []string{"apple", "banana"}
	vSlice := reflect.ValueOf(slice)
	fmt.Println("长度:", vSlice.Len())                // 输出: 长度: 2
	fmt.Println("第一个元素:", vSlice.Index(0).String()) // 输出: 第一个元素: apple

	// 修改元素（切片是可寻址的）
	vSlice.Index(1).SetString("cherry")
	fmt.Println("修改后:", slice) // 输出: 修改后: [apple cherry]

	// Map
	m := map[string]int{"a": 1, "b": 2}
	vMap := reflect.ValueOf(m)
	key := reflect.ValueOf("a")
	fmt.Println("值 'a':", vMap.MapIndex(key).Int()) // 输出: 值 'a': 1

	// 设置新键值
	vMap.SetMapIndex(reflect.ValueOf("c"), reflect.ValueOf(3))
	fmt.Println("修改后 Map:", m) // 输出: 修改后 Map: map[a:1 b:2 c:3]
}

type Calculator struct{}

func (c *Calculator) Add(a, b int) int {
	return a + b
}
func TestInvokeMethod(t *testing.T) {
	calc := &Calculator{}
	v := reflect.ValueOf(calc)

	// 调用方法
	method := v.MethodByName("Add")
	args := []reflect.Value{reflect.ValueOf(5), reflect.ValueOf(3)}
	result := method.Call(args)

	fmt.Println("结果:", result[0].Int()) // 输出: 结果: 8
}

func TestSetNewValue(t *testing.T) {
	// 创建 int 指针零值
	intType := reflect.TypeOf(0)
	newInt := reflect.New(intType)
	newInt.Elem().SetInt(100)
	fmt.Println("新值:", newInt.Elem().Int()) // 输出: 新值: 100

	// 创建切片
	sliceType := reflect.SliceOf(reflect.TypeOf(""))
	newSlice := reflect.MakeSlice(sliceType, 2, 2)
	newSlice.Index(0).SetString("hello")
	newSlice.Index(1).SetString("world")
	fmt.Println("新切片:", newSlice.Interface()) // 输出: 新切片: [hello world]

	// 创建 map
	mapType := reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(0))
	newMap := reflect.MakeMap(mapType)
	newMap.SetMapIndex(reflect.ValueOf("one"), reflect.ValueOf(1))
	newMap.SetMapIndex(reflect.ValueOf("two"), reflect.ValueOf(2))
	fmt.Println("新 Map:", newMap.Interface()) // 输出: 新 Map: map[one:1 two:2]

	// 创建 struct
	type Point struct {
		X, Y int
	}
	structType := reflect.TypeOf(Point{})
	newStruct := reflect.New(structType).Elem()
	newStruct.FieldByName("X").SetInt(10)
	newStruct.FieldByName("Y").SetInt(20)
	fmt.Println("新 Struct:", newStruct.Interface()) // 输出: 新 Struct: {10 20}

	// 创建函数
	funcType := reflect.TypeOf(func(a, b int) int { return 0 })
	newFunc := reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		a := args[0].Int()
		b := args[1].Int()
		return []reflect.Value{reflect.ValueOf(a + b)}
	})
	result := newFunc.Call([]reflect.Value{reflect.ValueOf(3), reflect.ValueOf(4)})
	fmt.Println("新函数调用结果:", result[0].Int()) // 输出: 新函数调用结果: 7

	// 创建接口
	interfaceType := reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	newInterface := reflect.New(interfaceType).Elem()
	fmt.Println("新接口:", newInterface.Interface()) // 输出: 新接口: <nil>

	// 创建Channel
	channelType := reflect.ChanOf(reflect.BothDir, reflect.TypeOf(""))
	newChannel := reflect.MakeChan(channelType, 0)
	fmt.Println("新 Channel:", newChannel.Interface()) // 输出: 新 Channel: 0xc0000a2000

}
