#!/bin/bash

#=====================================================
# Linux 系统优化脚本 - 游戏服务器专用
#=====================================================

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查是否为 root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "请使用 root 权限运行此脚本"
        echo "使用: sudo $0"
        exit 1
    fi
}

# 优化网络参数
optimize_network() {
    info "📡 优化网络参数..."

    # 备份原配置
    cp /etc/sysctl.conf /etc/sysctl.conf.bak.$(date +%Y%m%d)

    # 添加优化参数
    cat >> /etc/sysctl.conf << 'EOF'

# ================================================
# Virtual Router 游戏服务器网络优化
# ================================================

# TCP 连接队列
net.core.somaxconn = 8192
net.core.netdev_max_backlog = 16384
net.ipv4.tcp_max_syn_backlog = 8192

# TCP 参数优化
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 600
net.ipv4.tcp_keepalive_intvl = 10
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_max_tw_buckets = 5000

# 缓冲区优化
net.core.rmem_default = 262144
net.core.rmem_max = 16777216
net.core.wmem_default = 262144
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 262144 16777216
net.ipv4.tcp_wmem = 4096 262144 16777216

# 防止 SYN 攻击
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_orphans = 262144

# 快速回收 TIME_WAIT 连接
net.ipv4.tcp_timestamps = 1

EOF

    # 应用配置
    sysctl -p
    info "✅ 网络参数优化完成"
}

# 优化文件句柄限制
optimize_limits() {
    info "📂 优化文件句柄限制..."

    # 备份
    cp /etc/security/limits.conf /etc/security/limits.conf.bak.$(date +%Y%m%d)

    # 添加限制
    cat >> /etc/security/limits.conf << 'EOF'

# Virtual Router 游戏服务器限制
* soft nofile 655350
* hard nofile 655350
* soft nproc 655350
* hard nproc 655350

EOF

    # 修改 systemd 限制
    if [ -d /etc/systemd/system.conf.d ]; then
        mkdir -p /etc/systemd/system.conf.d
    fi

    cat > /etc/systemd/system.conf.d/limits.conf << 'EOF'
[Manager]
DefaultLimitNOFILE=655350
DefaultLimitNPROC=655350
EOF

    info "✅ 文件句柄限制优化完成"
    warn "⚠️ 需要重新登录或重启系统生效"
}

# 安装性能监控工具
install_monitoring() {
    info "📊 安装性能监控工具..."

    # 检测包管理器
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y htop iotop nethogs dstat sysstat
    elif command -v yum &> /dev/null; then
        yum install -y epel-release
        yum install -y htop iotop nethogs dstat sysstat
    else
        warn "未知的包管理器，请手动安装监控工具"
        return
    fi

    info "✅ 监控工具安装完成"
}

# 配置时区
set_timezone() {
    info "🌍 配置时区为 Asia/Shanghai..."
    timedatectl set-timezone Asia/Shanghai
    info "✅ 时区配置完成: $(date)"
}

# 禁用不必要的服务
disable_services() {
    info "🔧 禁用不必要的服务..."

    # 游戏服务器通常不需要的服务
    local services=("bluetooth" "cups" "avahi-daemon")

    for service in "${services[@]}"; do
        if systemctl is-active --quiet $service; then
            systemctl stop $service
            systemctl disable $service
            info "已禁用: $service"
        fi
    done

    info "✅ 服务优化完成"
}

# 优化内核参数
optimize_kernel() {
    info "🔧 优化内核参数..."

    cat >> /etc/sysctl.conf << 'EOF'

# 虚拟内存优化
vm.swappiness = 10
vm.dirty_ratio = 40
vm.dirty_background_ratio = 10

# 内核优化
kernel.pid_max = 4194303
kernel.threads-max = 4194303

EOF

    sysctl -p
    info "✅ 内核参数优化完成"
}

# 检测 io_uring 支持
check_io_uring() {
    info "🔍 检测 io_uring 支持..."

    KERNEL_VERSION=$(uname -r | cut -d. -f1)
    KERNEL_MINOR=$(uname -r | cut -d. -f2)

    if [ "$KERNEL_VERSION" -gt 5 ] || ([ "$KERNEL_VERSION" -eq 5 ] && [ "$KERNEL_MINOR" -ge 1 ]); then
        info "✅ 内核支持 io_uring ($(uname -r))"
        info "   Netty 将自动使用 io_uring 提升性能"
    else
        warn "⚠️ 内核不支持 io_uring ($(uname -r))"
        warn "   建议升级到 Linux 5.1+ 以获得最佳性能"
        warn "   性能提升: 吞吐量 +20-40%, CPU -15-25%"
    fi
}

# 创建应用目录
create_app_dirs() {
    info "📁 创建应用目录..."

    mkdir -p /opt/virtual-router
    mkdir -p /var/log/virtual-router

    # 设置权限
    chmod 755 /opt/virtual-router
    chmod 755 /var/log/virtual-router

    info "✅ 应用目录: /opt/virtual-router"
    info "✅ 日志目录: /var/log/virtual-router"
}

# 显示系统信息
show_system_info() {
    echo ""
    info "📊 系统信息"
    echo "========================================"

    # OS
    if [ -f /etc/os-release ]; then
        OS_NAME=$(grep "^NAME=" /etc/os-release | cut -d'"' -f2)
        OS_VERSION=$(grep "^VERSION=" /etc/os-release | cut -d'"' -f2)
        echo "操作系统: $OS_NAME $OS_VERSION"
    fi

    # 内核
    echo "内核版本: $(uname -r)"

    # CPU
    CPU_MODEL=$(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)
    CPU_CORES=$(nproc)
    echo "CPU: $CPU_MODEL"
    echo "CPU 核心: $CPU_CORES"

    # 内存
    TOTAL_MEM=$(free -h | grep "^Mem:" | awk '{print $2}')
    echo "总内存: $TOTAL_MEM"

    # 磁盘
    DISK_INFO=$(df -h / | tail -1 | awk '{print $2 " (已用 " $5 ")"}')
    echo "根分区: $DISK_INFO"

    echo "========================================"
    echo ""
}

# 主菜单
show_menu() {
    echo ""
    echo "=========================================="
    echo "  🎮 Linux 游戏服务器优化脚本"
    echo "=========================================="
    echo "1. 完整优化（推荐）"
    echo "2. 仅优化网络参数"
    echo "3. 仅优化文件句柄"
    echo "4. 安装监控工具"
    echo "5. 显示系统信息"
    echo "6. 检测 io_uring 支持"
    echo "0. 退出"
    echo "=========================================="
    echo -n "请选择 [0-6]: "
}

# 完整优化
full_optimize() {
    info "🚀 开始完整优化..."
    echo ""

    check_root
    show_system_info
    check_io_uring
    create_app_dirs
    optimize_network
    optimize_limits
    optimize_kernel
    set_timezone
    install_monitoring
    disable_services

    echo ""
    info "✅ 优化完成！"
    warn "⚠️ 某些优化需要重新登录或重启系统生效"
    echo ""
    info "下一步:"
    info "1. 重启系统: sudo reboot"
    info "2. 上传应用: python upload_to_server.py"
    info "3. 启动服务: cd /opt/virtual-router && ./restart-virtual-router-center.sh start"
}

# 主程序
main() {
    if [ "$1" == "--auto" ]; then
        # 自动模式
        full_optimize
        exit 0
    fi

    # 交互模式
    while true; do
        show_menu
        read -r choice

        case $choice in
            1)
                full_optimize
                break
                ;;
            2)
                check_root
                optimize_network
                ;;
            3)
                check_root
                optimize_limits
                ;;
            4)
                check_root
                install_monitoring
                ;;
            5)
                show_system_info
                ;;
            6)
                check_io_uring
                ;;
            0)
                info "退出"
                exit 0
                ;;
            *)
                error "无效选择"
                ;;
        esac
    done
}

main "$@"


