#!/bin/bash
address=$1
NamespaceId=$2
Group=$3
DataId=$4
echo "$address"
echo "$NamespaceId"
echo "$Group"
echo "$DataId"

url=$(curl -F file=@./ctp-device-server  http://10.225.20.184:11001/group1/upload)
control_url=$(curl -F file=@./control.sh  http://10.225.20.184:11001/group1/upload)
echo ">>>>>>>>>>>>>>>>>"
echo "$url"
echo "$control_url"
echo ">>>>>>>>>>>>>>>>>"
mkdir "/root/.ssh"
touch "/root/.ssh/known_hosts"
chmod 600 "/root/.ssh"
chmod 600 "/root/.ssh/known_hosts"
echo "10.225.22.129 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBIXeDezlsEFWcoG4pxFgu31g3+sNX59/x32lSMcqAVdL2uyaOR4WOt8i+MIympDfV4jBrkH1hYGxbfwUdpqi1ec=" >> ~/.ssh/known_hosts

#重启主站
sshpass -p youQAka ssh -tt root@"$address" <<EOF

cd /home/yuzhiming/server/device-server-go
mv -f ctp-device-server ctp-device-server-cp
mv -f control.sh control.sh-cp
wget -O /home/yuzhiming/server/device-server-go/ctp-device-server $url --debug
wait
wget -O /home/yuzhiming/server/device-server-go/control.sh $control_url --debug
chmod +x control.sh
chmod +x ctp-device-server
export NamespaceId="$NamespaceId"
export Group="$Group"
export DataId="$DataId"
sh ./control.sh restart
exit
EOF
echo success