// Use of this source code is governed by a Apache-style
// license that can be found in the LICENSE file.

package onnx

/*
#cgo LDFLAGS: -lonnxruntime -lm
#cgo CFLAGS: -O3
#cgo arm64 CFLAGS: -I../../lib/arm64/onnxruntime/1.14.0/include -DCOREML=1
#cgo amd64 CFLAGS: -I../../lib/amd64/onnxruntime/1.14.0/include
#cgo amd64 LDFLAGS: -L../../lib/amd64/onnxruntime/1.14.0/lib
#include "onnx_capi.h"
*/
import "C"
import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

type Model struct {
	env        *C.OnnxEnv
	inputNames []string
}

type Tensor struct {
	ortValue *C.OrtValue
}

type ExecutionProvider int

const (
	ModeCPU      ExecutionProvider = iota // 0
	ModeCUDA                              // 1
	ModeTensorRT                          // 2
	ModeCoreML                            // 3
)

func NewModel(
	model_path string,
	inputNames []string,
	outputNames []string,
	mode ExecutionProvider) *Model {

	ptr := C.CString(model_path)
	defer C.free(unsafe.Pointer(ptr))

	session := C.OnnxNewOrtSession(ptr, C.int(mode))

	session.input_names_len = C.size_t(len(inputNames))
	for i, s := range inputNames {
		session.input_names[i] = C.CString(s)
	}

	session.output_names_len = C.size_t(len(outputNames))
	for i, s := range outputNames {
		session.output_names[i] = C.CString(s)
	}

	return &Model{
		env:        session,
		inputNames: inputNames,
	}
}

func (this *Model) NewInt64Tensor(dims []int64, values []int64) *Tensor {
	ortValue := C.OnnxCreateTensorInt64(
		this.env,
		(*C.int64_t)(unsafe.Pointer(&values[0])),
		C.size_t(len(values)*8),
		(*C.int64_t)(unsafe.Pointer(&dims[0])),
		C.size_t(len(dims)))
	return &Tensor{ortValue: ortValue}
}

func (this *Model) NewFloat32Tensor(dims []int64, values []float32) *Tensor {
	ortValue := C.OnnxCreateTensorFloat32(
		this.env,
		(*C.float)(unsafe.Pointer(&values[0])),
		C.size_t(len(values)*8),
		(*C.int64_t)(unsafe.Pointer(&dims[0])),
		C.size_t(len(dims)))
	return &Tensor{ortValue: ortValue}
}

// Invoke the task.
func (this *Model) RunInference(data map[string]*Tensor) []*Tensor {
	inputs := make([]*C.OrtValue, len(this.inputNames))
	for i, name := range this.inputNames {
		tensor, ok := data[name]
		if !ok {
			panic(fmt.Sprintf("input %s not found", name))
		}

		inputs[i] = tensor.ortValue
	}

	outputs := make([]*C.OrtValue, this.env.output_names_len)

	C.OnnxRunInference(this.env,
		(**C.OrtValue)(unsafe.Pointer(&inputs[0])),
		(**C.OrtValue)(unsafe.Pointer(&outputs[0])))

	outputTensors := make([]*Tensor, this.env.output_names_len)
	for i := 0; i < int(this.env.output_names_len); i++ {
		outputTensors[i] = &Tensor{ortValue: outputs[i]}
	}

	return outputTensors
}

func (this *Model) Delete() {
	if this != nil {
		C.OnnxDeleteOrtSession(this.env)
	}
}

func (this *Tensor) NumDims() int {
	return int(C.OnnxTensorNumDims(this.ortValue))
}

// Dim return dimension of the element specified by index.
func (this *Tensor) Dim(index int) int64 {
	return int64(C.OnnxTensorDim(this.ortValue, C.int32_t(index)))
}

// Shape return shape of the tensor.
func (this *Tensor) Shape() []int64 {
	shape := make([]int64, this.NumDims())
	for i := 0; i < this.NumDims(); i++ {
		shape[i] = this.Dim(i)
	}
	return shape
}

func (this *Tensor) Size() int64 {
	shape := this.Shape()
	x := int64(1)
	for _, s := range shape {
		x *= s
	}
	return x
}

func (this *Tensor) Delete() {
	if this != nil {
		C.OnnxReleaseTensor(this.ortValue)
	}
}

func (this *Tensor) CopyToBuffer(b interface{}, size int) {
	C.OnnxTensorCopyToBuffer(this.ortValue, unsafe.Pointer(reflect.ValueOf(b).Pointer()), C.size_t(size))
}

var EuclideanDistance512 = func(d [][]float32, ai, bi, end int) []float32 {
	var (
		s, t float32
	)
	res := make([]float32, end-bi)
	c := 0
	for j := bi; j < end; j++ {
		s = 0
		t = 0
		for i := 0; i < 512; i++ {
			t = d[ai][i] - d[j][i]
			s += t * t
		}

		res[c] = float32(math.Sqrt(float64(s)))

		c++
	}
	return res
}

var EuclideanDistance512C = func(d [][]float32, ai, bi, end int) []float32 {
	res := make([]float32, end-bi)

	data := C.MakeFloatArray(C.int(len(d)))
	defer C.FreeFloatArray(data)
	for i, v := range d {
		C.SetFloatArray(data, (*C.float)(unsafe.Pointer(&v[0])), C.int(i))
	}

	C.EuclideanDistance512(
		data,
		(*C.float)(unsafe.Pointer(&res[0])),
		C.int(ai),
		C.int(bi),
		C.int(end),
	)
	return res
}
