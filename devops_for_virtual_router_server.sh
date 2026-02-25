#!/usr/bin/env sh

set -eu

APP_NAME="virtual-router-server"
APP_HOME="$(cd "$(dirname "$0")" && pwd)"
APP_BIN="$APP_HOME/$APP_NAME"
APP_CONF="$APP_HOME/neko233-router-server.json"
APP_LOG_DIR="$APP_HOME/logs"
APP_LOG_FILE="$APP_LOG_DIR/$APP_NAME.out.log"
APP_PID_FILE="$APP_HOME/$APP_NAME.pid"

mkdir -p "$APP_LOG_DIR"

is_running() {
	if [ -f "$APP_PID_FILE" ]; then
		pid="$(cat "$APP_PID_FILE")"
		if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
			return 0
		fi
	fi
	return 1
}

start() {
	if [ ! -f "$APP_BIN" ]; then
		echo "[$APP_NAME] 可执行文件不存在: $APP_BIN"
		exit 1
	fi

	if [ ! -x "$APP_BIN" ]; then
		echo "[$APP_NAME] 检测到无执行权限，自动修复: chmod +x $APP_BIN"
		chmod +x "$APP_BIN"
	fi

	if [ ! -x "$APP_BIN" ]; then
		echo "[$APP_NAME] 可执行权限修复失败，请手动执行: chmod +x $APP_BIN"
		exit 1
	fi

	if [ ! -f "$APP_CONF" ]; then
		echo "[$APP_NAME] 配置文件不存在: $APP_CONF"
		exit 1
	fi

	if is_running; then
		echo "[$APP_NAME] 已在运行, pid=$(cat "$APP_PID_FILE")"
		return
	fi

	echo "[$APP_NAME] 启动中..."
	nohup "$APP_BIN" >> "$APP_LOG_FILE" 2>&1 &
	echo $! > "$APP_PID_FILE"
	sleep 1

	if is_running; then
		echo "[$APP_NAME] 启动成功, pid=$(cat "$APP_PID_FILE")"
	else
		echo "[$APP_NAME] 启动失败，请检查日志: $APP_LOG_FILE"
		rm -f "$APP_PID_FILE"
		exit 1
	fi
}

stop() {
	if ! is_running; then
		echo "[$APP_NAME] 未运行"
		rm -f "$APP_PID_FILE"
		return
	fi

	pid="$(cat "$APP_PID_FILE")"
	echo "[$APP_NAME] 停止中, pid=$pid"
	kill "$pid" >/dev/null 2>&1 || true

	i=0
	while [ $i -lt 10 ]; do
		if ! kill -0 "$pid" >/dev/null 2>&1; then
			break
		fi
		sleep 1
		i=$((i + 1))
	done

	if kill -0 "$pid" >/dev/null 2>&1; then
		echo "[$APP_NAME] 进程未退出，执行强制停止"
		kill -9 "$pid" >/dev/null 2>&1 || true
	fi

	rm -f "$APP_PID_FILE"
	echo "[$APP_NAME] 已停止"
}

status() {
	if is_running; then
		echo "[$APP_NAME] 运行中, pid=$(cat "$APP_PID_FILE")"
	else
		echo "[$APP_NAME] 未运行"
	fi
}

logs() {
	touch "$APP_LOG_FILE"
	tail -n 200 -f "$APP_LOG_FILE"
}

restart() {
	stop
	start
}

usage() {
	cat <<EOF
用法: $0 {start|stop|restart|status|logs}
EOF
}

case "${1:-}" in
	start)
		start
		;;
	stop)
		stop
		;;
	restart)
		restart
		;;
	status)
		status
		;;
	logs)
		logs
		;;
	*)
		usage
		exit 1
		;;
esac

