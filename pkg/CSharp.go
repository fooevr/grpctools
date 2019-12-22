package pkg

import (
	"fmt"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"log"
	"path/filepath"
	"strings"
)

type File struct {
	FileName     string
	FullFileName string
	Messages     []*Message
	CsNamespace  string
	JavaPackage  string
	PhpNamespace string
	GoPackage    string
	Comment      []string
}

type Message struct {
	Name         string
	csNamespace  string
	javaPackage  string
	phpNamespace string
	goPackage    string
	Fields       []*Field
	desc         *desc.MessageDescriptor
	Comment      []string
}

type FieldType int

const (
	FieldType_Message = 0
	FieldType_Value = 1
	FieldType_Map = 2
	FieldType_Repeated = 3
)

type Field struct {
	Name      string
	CsType    string
	JavaType  string
	GoType    string
	PhpType   string
	SwiftType string
	ObjcType  string

	CsKeyType    string
	JavaKeyType  string
	GoKeyType    string
	PhpKeyType   string
	SwiftKeyType string
	ObjcKeyType  string

	FieldType FieldType
	Comment   []string
	Nullable	bool
}

func GenerateCSharp(files map[string]*desc.FileDescriptor, language string) []*File {
	result := []*File{}
	fqns := map[string]*Message{}
	for file, fileDesc := range files{
		file := &File{
			FileName:     filepath.Base(file),
			FullFileName: file,
			Messages:     []*Message{},
			CsNamespace:  fileDesc.GetFileOptions().GetCsharpNamespace(),
			JavaPackage:  fileDesc.GetFileOptions().GetJavaPackage(),
			PhpNamespace: fileDesc.GetFileOptions().GetPhpNamespace(),
			GoPackage:    fileDesc.GetFileOptions().GetGoPackage(),
			Comment:	  getComment(fileDesc.GetSourceInfo()),
		}
		if file.CsNamespace == "" {
			file.CsNamespace = fileDesc.GetPackage()
		}
		if file.JavaPackage == "" {
			file.JavaPackage = fileDesc.GetPackage()
		}
		if file.PhpNamespace == "" {
			file.PhpNamespace = fileDesc.GetPackage()
		}
		if file.GoPackage == "" {
			file.GoPackage = fileDesc.GetPackage()
		}

		for _, messageDesc := range fileDesc.GetMessageTypes(){
			msg := &Message{
				Name:         messageDesc.GetName(),
				Fields:       []*Field{},
				desc:         messageDesc,
				csNamespace:  file.CsNamespace,
				javaPackage:  file.JavaPackage,
				phpNamespace: file.PhpNamespace,
				goPackage:    file.GoPackage,
				Comment:	  getComment(messageDesc.GetSourceInfo()),
			}
			file.Messages = append(file.Messages, msg)
			fqns[messageDesc.GetFullyQualifiedName()] = msg
		}
		result = append(result, file)
	}
	for _, file := range result{
		for _, msg := range file.Messages {
			for _, fieldDesc := range msg.desc.GetFields(){
				field := &Field{
					Name: fieldDesc.GetName(),
					Comment:	  getComment(fieldDesc.GetSourceInfo()),
				}
				if fieldDesc.IsMap() {
					switch language {
					case "cs":
						field.CsKeyType = getFieldTypeName(fieldDesc.GetMapKeyType(), fqns, language)
						field.CsType = getFieldTypeName(fieldDesc.GetMapValueType(), fqns, language)
					}
					field.FieldType = FieldType_Map
				}else{
					if fieldDesc.IsRepeated() {
						field.FieldType = FieldType_Repeated
					}else if fieldDesc.GetMessageType()!=nil{
						if fieldDesc.GetMessageType().GetFile().GetPackage() == "com.google.protobuf" {
							field.FieldType = FieldType_Value
							field.Nullable = true
						}else {
							field.FieldType = FieldType_Message
						}
					}else{
						field.FieldType = FieldType_Value
					}

					switch language {
					case "cs":
						field.CsType = getFieldTypeName(fieldDesc, fqns, language)
					}
				}
				msg.Fields = append(msg.Fields, field)
			}
		}
	}
	return result
}

var typeMap = map[descriptor.FieldDescriptorProto_Type]map[string]string{
	descriptor.FieldDescriptorProto_TYPE_BOOL: {"cs": "bool"},
	descriptor.FieldDescriptorProto_TYPE_BYTES: {"cs": "byte[]"},
	descriptor.FieldDescriptorProto_TYPE_DOUBLE: {"cs": "double"},
	descriptor.FieldDescriptorProto_TYPE_FLOAT: {"cs": "float"},
	descriptor.FieldDescriptorProto_TYPE_INT32: {"cs": "int32"},
	descriptor.FieldDescriptorProto_TYPE_INT64: {"cs": "int64"},
	descriptor.FieldDescriptorProto_TYPE_STRING: {"cs": "string"},
}

func getFieldTypeName(fieldDesc *desc.FieldDescriptor, fqns map[string]*Message, language string)string{
	if fieldDesc.GetMessageType() != nil{
		if fieldDesc.GetMessageType().GetFile().GetPackage() == "com.google.protobuf"{
			return typeMap[fieldDesc.GetMessageType().GetFields()[0].GetType()][language]
		}else {
			msg := fqns[fieldDesc.GetMessageType().GetFullyQualifiedName()]
			switch language {
			case "cs":
				return fmt.Sprintf("%s.%s", msg.csNamespace, msg.Name)
			}
		}
	}else{
		return typeMap[fieldDesc.GetType()][language]
	}
	log.Fatal("unsupport field type", fieldDesc.GetType())
	return ""
}

func getComment(si *descriptor.SourceCodeInfo_Location)[]string{
	result := []string{}
	cms := strings.Split(strings.Trim(strings.ReplaceAll(si.GetLeadingComments(), "\r\n", "\n"), "\n"), "\n")
	for _, item := range cms{
		result = append(result, item)
	}
	result = append(result, strings.Trim(strings.ReplaceAll(si.GetTrailingComments(), "\r\n", "\n"), "\n"))
	return result
}