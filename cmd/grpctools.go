package main

import (
	"encoding/base64"
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"grpctools/pkg"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	pflag.StringP("dir", "d", "", "proto文件夹根目录")
	pflag.StringP("out", "o", "", "DAO文件输出路径")
	pflag.StringP("language", "l", "", "语言: java, cs, js, kotlin, swift, oc")
	pflag.StringP("suffix", "s", "", "命名空间后缀")
	pflag.BoolP("help", "h", false, "参数说明")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	if len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		pflag.Usage()
		return
	}
	if viper.GetString("dir") == "" {
		fmt.Println("proto根目录不能为空")
		return
	}

	if viper.GetString("out") == "" {
		fmt.Println("请指定DAO输出目录")
		return
	}
	if viper.GetString("language") == "" {
		fmt.Println("请指定输出语言")
		return
	}

	wrapperFs := new(descriptor.FileDescriptorSet)
	wrapperByte, _ := base64.StdEncoding.DecodeString("Cv4DCh5nb29nbGUvcHJvdG9idWYvd3JhcHBlcnMucHJvdG8SD2dvb2dsZS5wcm90b2J1ZiIjCgtEb3VibGVWYWx1ZRIUCgV2YWx1ZRgBIAEoAVIFdmFsdWUiIgoKRmxvYXRWYWx1ZRIUCgV2YWx1ZRgBIAEoAlIFdmFsdWUiIgoKSW50NjRWYWx1ZRIUCgV2YWx1ZRgBIAEoA1IFdmFsdWUiIwoLVUludDY0VmFsdWUSFAoFdmFsdWUYASABKARSBXZhbHVlIiIKCkludDMyVmFsdWUSFAoFdmFsdWUYASABKAVSBXZhbHVlIiMKC1VJbnQzMlZhbHVlEhQKBXZhbHVlGAEgASgNUgV2YWx1ZSIhCglCb29sVmFsdWUSFAoFdmFsdWUYASABKAhSBXZhbHVlIiMKC1N0cmluZ1ZhbHVlEhQKBXZhbHVlGAEgASgJUgV2YWx1ZSIiCgpCeXRlc1ZhbHVlEhQKBXZhbHVlGAEgASgMUgV2YWx1ZUJ8ChNjb20uZ29vZ2xlLnByb3RvYnVmQg1XcmFwcGVyc1Byb3RvUAFaKmdpdGh1Yi5jb20vZ29sYW5nL3Byb3RvYnVmL3B0eXBlcy93cmFwcGVyc/gBAaICA0dQQqoCHkdvb2dsZS5Qcm90b2J1Zi5XZWxsS25vd25UeXBlc2IGcHJvdG8z")
	wrapperFs.XXX_Unmarshal(wrapperByte)

	fileDescs := []*desc.FileDescriptor{}
	parser := protoparse.Parser{IncludeSourceCodeInfo: true, ImportPaths: []string{viper.GetString("dir")}}
	filepath.Walk(viper.GetString("dir"), func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			relPath, err := filepath.Rel(viper.GetString("dir"), path)
			if err != nil {
				fmt.Println(err)
			}
			d, err := parser.ParseFiles(relPath)
			if err != nil {
				fmt.Println(err)
				fmt.Println("解析proto文件出错。可能是因为文本编码问题，windows上应为GB2312. visual studio中打开高级保存选项，设置文件为GB2312代码页936.")
				return nil
			}
			fileDescs = append(fileDescs, d[0])
		}
		return nil
	})
	sysFiles, _ := desc.CreateFileDescriptorsFromSet(wrapperFs)
	for _, f := range sysFiles {
		fileDescs = append(fileDescs, f)
	}

	outputs := map[string]*desc.FileDescriptor{}
	for _, fileDesc := range fileDescs {
		rootDir := viper.GetString("out")
		paths := strings.Split(strings.TrimSuffix(fileDesc.GetName(), ".proto"), string(os.PathSeparator))
		dir := filepath.Join(rootDir, filepath.Join(paths[0:len(paths)-1]...))
		fullPath := filepath.Join(dir, string(os.PathSeparator), paths[len(paths)-1]+".cs")
		if !strings.HasPrefix(fileDesc.GetPackage(), "google.protobuf") {
			os.MkdirAll(dir, 0700)
			x, _ := os.Create(fullPath)
			x.Close()
		}
		outputs[fullPath] = fileDesc
	}
	metas := pkg.GenerateCSharp(outputs, viper.GetString("language"))
	//temp, err := template.ParseFiles("./template/cs.gohtml")
	ptemp, err := pongo2.FromFile("./template/cs.temp")
	if err != nil {
		fmt.Println(err)
	}
	for _, fileArg := range metas {
		if strings.Contains(fileArg.FullFileName, "/google/protobuf/") {
			continue
		}
		os.Remove(fileArg.FullFileName)
		fi, err := os.Create(fileArg.FullFileName)
		if err != nil {
			log.Panic(err)
			return
		}
		ptemp.Options.Update(&pongo2.Options{TrimBlocks: true, LStripBlocks: true})
		ptemp.ExecuteWriter(pongo2.Context{"data": fileArg}, fi)
		fi.Close()
	}
}
