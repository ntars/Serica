# D-0184 整数映射删除边界

## 决策

新增语句 `删映射键（映射，整数键）`，要求 `map.write`；沿用有符号 64 位整数键值映射的桶链布局。删除命中后把后继挂到前驱槽，并把旧条目键替换为墓哨，查找跳过墓哨、插入仍可复用条目存储。空键和墓哨均不作为用户键参与查找，避免与哨兵冲突。

## 验证

`make test-p1b-runtime` 通过 209 个 arm64 整数形态产物，其中 13 个映射产物覆盖删除存在键、缺失键、删除后复用、碰撞链保持和扩容后删除；新增 5 个编译诊断反例。`make seed-audit`、`test-p1b-static`、`test-p1b-modules`、`test-p1b-module-effect`、`test-p1b-module-type`、`test-p1b-native-emit` 均通过。P1b 阶段门仍未通过。
