# gosync  
A sync tools by golang. Sync files from one node to one or more nodes.  

## 算法流程   
1. 在每个主机节点运行守护进程gosync;  
2. 在任意主机上打开客户端, 向需要向其他主机同步文件的主机发起同步任务;  
3. 源主机向所有目标主机发起连接, 即第一层连接;  
4. 源主机向所有目标主机发送本地同步文件列表和列表的md5;  
5. 目标主机收到文件列表和本地文件列表进行对比(对比的是md5);  
6. 目标主机对本地目录, 软连接, 需要删除的文件进行创建或删除操作;  
7. 目标主机把需要同步的文件列表返回给源主机;  
8. 源主机收到所有目标主机的请求列表, 进行比对归类;  
9. 源主机调用N个goroutine对每个类别并发执行同步任务;  
10. 在每个类别下, 建立起一个第二层连接, 二层连接是一个树状结构的连接树;  
11. 在每个类别下同步文件;  
12. 每个二层连接中的节点都会把接收到的数据在保存到本地的同时向下游节点转发;
13. 当二层连接的节点发现自己的同步任务已完成时, 会把结果发送到一层连接;  
14. 二层连接被断开;  
15. 一层连接将同步结果返回给源主机;  
16. 源主机将结果汇总返回给客户端;  
17. 一层网络被断开, 任务执行完成;  
18. 守护进程继续等待下次的同步任务;  

### 两层连接模型   

### 第一层连接   
源主机节点和所有的目标主机节点建立连接;  
用于传输源文件列表, 返回请求文件列表, 及同步结果.  
![chart1](https://github.com/jacenr/gosync/blob/alg2/Screenshots/gosync_p1.png)  
### 第二层连接   
按类别建立起连接树;  
用于传输文件数据流.  
![chart1](https://github.com/jacenr/gosync/blob/alg2/Screenshots/gosync_p2.png)  

### 使用帮助   
1. 编译服务端文件和客户端文件;  
```
[root@QCLOUD_KOK_10 gosync]# go build main/gosyncClient.go 
[root@QCLOUD_KOK_10 gosync]# go build main/gosyncServer.go
```
2. 在所有主机节点运行服务端程序;  
```
[root@QCLOUD_KOK_10 jacenr]# ./gosyncServer
```
3. 在源节点执行客户端程序;  
```
[root@QCLOUD_KOK_10 jacenr]# ./gosyncClient -t="10.10.30.207,10.10.30.208,10.10.30.209" -dst="/tmp/testdst" -src="/tmp/testsrc" -d=true
```
	说明:  
		-t string 目标主机列表(后期考虑从文件中读取列表)  
		-dst string 目标主机目录  
		-src string 源主机目录  
		-d bool 删除源主机目录中不存在的文件和目录  
		-z bool 是否启用zip压缩, 节省流量, 加快传输  
			此选项未在示例中使用  
4. 执行结果:
```
同步前:
源主机
[root@QCLOUD_KOK_10 tmp]# tree testsrc
testsrc
├── d1
│   ├── f1s -> f1.txt
│   ├── f1.txt
│   ├── f2.txt
│   ├── f3s -> f3.txt
│   └── f3.txt
├── d2
├── f3s -> f3.txt
├── f3.txt
├── f4.txt
├── f5s -> f5.txt
└── f5.txt
目标主机
[root@QCLOUD_KOK_10 tmp]# tree testdst
testdst
├── d1
│   ├── f1s -> f1.txt
│   ├── f1.txt
│   ├── f2.txt
│   ├── f4s -> f4.txt
│   └── f4.txt
├── d3
├── f3s -> f3.txt
├── f3.txt
├── f4.txt
├── f6s -> f6.txt
└── f6.txt

同步后:
目标主机
[root@QCLOUD_KOK_10 tmp]# tree testdst
testdst
├── d1
│   ├── f1s -> f1.txt
│   ├── f1.txt
│   ├── f2.txt
│   ├── f3s -> f3.txt
│   └── f3.txt
├── d2
├── f3s -> f3.txt
├── f3.txt
├── f4.txt
├── f5s -> f5.txt
└── f5.txt

```