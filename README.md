# itools 网络检测工具站点

## 功能简介
- **Ping检测**（ICMP/TCP fallback）
- **网站测速**（HTTP下载速度）
- **TCPing检测**（端口连通性及时延）
- **路由追踪**（traceroute）
- **DNS记录查询**（A/MX/NS/TXT等）
- **批量Ping**
- **批量HTML(S)测试**（可检测响应码/关键字/速度）
- **后台管理系统**（建议用 Directus 或 Strapi，支持广告管理、接口配置、权限管理等）

## 技术架构建议
- 后端API：Go（Gin框架），高并发/易扩展
- 前端：Next.js（React），支持自定义UI和API对接
- 管理后台：Directus（开源CMS，支持内容、广告、探针、接口Key等管理）
- 数据库：PostgreSQL
- 缓存/队列：Redis
- 容器化：Docker+docker-compose（本地开发）

## 快速启动
1. `docker-compose up --build`
2. 访问
   - 管理后台 http://localhost:8055
   - API接口 http://localhost:8080
   - 前端页面 http://localhost:3000

## 安全建议
- API鉴权（Key或JWT）
- 速率限制和作业队列
- 日志、监控（Prometheus/Grafana）
- 合规使用，防滥用

---

如需上传其它文件，请回复对应序号或名称，我会帮你生成内容。