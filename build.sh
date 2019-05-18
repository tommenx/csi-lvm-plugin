# !/usr/bin/env bash
# 非0返回值，立即退出脚本执行
set -e

cd ${GOPATH}/src/github.com/tommenx/csi-lvm-plugin/cmd

# 测试代码，编辑本机版本
go build -o lvmplugin.csi.alibabacloud.com
mv lvmplugin.csi.alibabacloud.com ../output/ 
