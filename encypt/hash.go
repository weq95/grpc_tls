package encypy

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

//一致性hash算法
//当 hash 环上没有数据时提示错误
var errEmpty = errors.New("Hash 环没有数据")

type uints []uint32

// Len 返回切片长度
func (x uints) Len() int {
	return len(x)
}

// Less 比较两个数的大小
func (x uints) Less(i, j int) bool {
	return x[i] < x[j]
}

// Swap 切片中两个值进行交换
func (x uints) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

// Consistent 创建结构体保存一致性Hash信息
type Consistent struct {
	//hash环 key为hash值 值为存放节点的信息
	circle map[uint32]string
	//已经排序的节点hash切片
	sortedHashes uints
	//虚拟节点个数，用来增加hash的平衡性
	VirtualNode int
	//map读写锁，保证高并发读取数据不会错误
	sync.RWMutex
}

// NewConsistent 创建一致性 hash 算法结构体， 设置默认节点数量
func NewConsistent() *Consistent {
	return &Consistent{
		circle: make(map[uint32]string, 0),
		//设置虚拟节点个数
		VirtualNode: 20,
	}
}

//自动生产key，根据服务器信息加上index，生产服务器相关的key
func (c *Consistent) generateKey(element string, index int) string {
	//副本key生产逻辑

	return element + strconv.Itoa(index)
}

//获取hash位置, 根据传入的key获取相应的hash值
func (c *Consistent) hashKey(key string) uint32 {
	if len(key) < 64 {
		var srcatch [64]byte

		//拷贝数据到数组中
		copy(srcatch[:], key)

		//使用IEEE 多项式返回数据的 CRC-32 校验和
		//是一项标准， 帮助我们通过算法算出key对应的hash值
		return crc32.ChecksumIEEE(srcatch[:len(key)])
	}

	return crc32.ChecksumIEEE([]byte(key))
}

//更新排序，方便查找
func (c *Consistent) updateSortedHashes() {
	//清空切片中的内容
	hashes := c.sortedHashes[:0]

	//判断容量是否过大，如果过大则重置
	//用于排序的切片的容量除以虚拟节点个数的4倍， 大于hash环上的节点数，则清空
	if cap(c.sortedHashes)/(c.VirtualNode*4) > len(c.circle) {
		hashes = nil
	}

	//添加hashs
	for k := range c.circle {
		hashes = append(hashes, k)
	}

	//对所有节点hash值进行排序， 方便进行二分法查找
	sort.Sort(hashes)

	//重新赋值
	c.sortedHashes = hashes
}

// Add 动态添加节点
func (c *Consistent) Add(element string) {
	c.Lock()

	c.add(element)

	c.Unlock()
}

//添加节点， 向hash环添加服务器信息
func (c *Consistent) add(element string) {
	//虚幻黄色至节点， 设置副本
	//虚拟节点上generateKey()规则生成虚拟节点信息， 根据虚拟节点信息进行hash
	for i := 0; i < c.VirtualNode; i++ {
		//根据虚拟节点添加到hash环中
		c.circle[c.hashKey(c.generateKey(element, i))] = element
	}

	//对生成的虚拟节点进行排序
	c.updateSortedHashes()
}

// Remove 动态删除节点
func (c *Consistent) Remove(element string) {
	c.Lock()
	defer c.Unlock()

	c.remove(element)
}

//删除节点， 需要传入服务器的信息， 比如IP地址
func (c *Consistent) remove(element string) {
	//根据传入的服务器的信息删除所有的副节点
	for i := 0; i < c.VirtualNode; i++ {
		delete(c.circle, c.hashKey(c.generateKey(element, i)))
	}

	//进行排序，方便二分查找
	c.updateSortedHashes()
}

//顺时针查找最近的节点
func (c *Consistent) search(key uint32) int {
	//查找算法
	i := sort.Search(len(c.sortedHashes), func(x int) bool {
		return c.sortedHashes[x] > key
	})

	//如果key超出范围则设置为0
	if i >= len(c.sortedHashes) {
		i = 0
	}

	return i
}

// Get 根据标识获取最近的服务器节点信息， 返回的是IP地址
func (c *Consistent) Get(name string) (string, error) {
	c.RLock()
	defer c.RLocker()

	//如果为0则返回错误
	if len(c.circle) == 0 {
		return "", errEmpty
	}

	//计算hash值
	var (
		key = c.hashKey(name)
		i   = c.search(key)
	)

	return c.circle[c.sortedHashes[i]], nil
}
