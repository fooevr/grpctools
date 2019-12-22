package main

import (
	"fmt"
	"github.com/flosch/pongo2"
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
	pflag.BoolP("help", "h", false, "参数说明")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	if len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" ) {
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
	//files := []string{}
	fileDescs := []*desc.FileDescriptor{}
	parser := protoparse.Parser{IncludeSourceCodeInfo:true, ImportPaths: []string{viper.GetString("dir")}}
	filepath.Walk(viper.GetString("dir"), func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			relPath, err := filepath.Rel(viper.GetString("dir"), path)
			if err != nil {
				fmt.Println(err)
			}
			d, err := parser.ParseFiles(relPath)
			if err != nil{
				fmt.Println(err)
				fmt.Println("解析proto文件出错。可能是因为文本编码问题，windows上应为GB2312. visual studio中打开高级保存选项，设置文件为GB2312代码页936.")
				return nil
			}
			if d[0].GetPackage() == "com.google.protobuf" {
				return nil
			}
			fileDescs = append(fileDescs, d[0])
		}
		return nil
	})

	outputs := map[string]*desc.FileDescriptor{}
	for _, fileDesc := range fileDescs{
		rootDir := viper.GetString("out")
		paths := strings.Split(strings.TrimSuffix(fileDesc.GetName(), ".proto"), string(os.PathSeparator))
		dir := filepath.Join(rootDir, filepath.Join(paths[0:len(paths) - 1]...))
		os.MkdirAll(dir, os.ModeDir)
		fullPath := filepath.Join(dir, string(os.PathSeparator), paths[len(paths) - 1] + ".cs")
		os.Create(fullPath)
		outputs[fullPath] = fileDesc
	}
	metas := pkg.GenerateCSharp(outputs, viper.GetString("language"))
	//temp, err := template.ParseFiles("./template/cs.gohtml")
	ptemp, err := pongo2.FromFile("./template/cs.temp")
	if err != nil{
		fmt.Println(err)
	}
	for _, fileArg := range metas{
		os.Remove(fileArg.FullFileName)
		fi, err := os.Create(fileArg.FullFileName)
		if err != nil{
			log.Panic(err)
			return
		}
		//temp.Execute(fi, fileArg)
		ptemp.Options.Update(&pongo2.Options{TrimBlocks:true, LStripBlocks:true})
		ptemp.ExecuteWriter(pongo2.Context{"data": fileArg}, fi)
		fi.Close()
	}
}
