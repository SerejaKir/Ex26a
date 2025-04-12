package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Задание: пайплайн, работающий с целыми числами

// Интервал очистки кольцевого буфера
const bufferDrainInterval time.Duration = 12 * time.Second

// Размер кольцевого буфера
const bufferSize int = 10

// RingIntBuffer - кольцевой буфер целых чисел
type RingIntBuffer struct {
	array []int // более низкоуровневое хранилище нашего
	// буфера
	pos  int        // текущая позиция кольцевого буфера
	size int        // общий размер буфера
	m    sync.Mutex // мьютекс для потокобезопасного доступа к
	// буферу.
	// Исключительный доступ нужен,
	// так так одновременно может быть вызваны
	// методы Get и Push,
	// первый - когда настало время вывести
	// содержимое буфера и очистить его,
	// второй - когда пользователь ввел новое
	// число, оба события обрабатываются разными
	// горутинами.

}

// NewRingIntBuffer - создание нового буфера целых чисел
func NewRingIntBuffer(size int) *RingIntBuffer {
	fmt.Println("[LOG] Создание нового кольцевого буфера размером", size)
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

// Push добавление нового элемента в конец буфера
// При попытке добавления нового элемента в заполненный буфер
// самое старое значение затирается
func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	fmt.Printf("[LOG] Добавление элемента %d в буфер (текущая позиция: %d)\n", el, r.pos)
	if r.pos == r.size-1 {
		// Сдвигаем все элементы буфера
		// на одну позицию в сторону начала
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

// Get - получение всех элементов буфера и его последующая очистка
func (r *RingIntBuffer) Get() []int {
	if r.pos < 0 {
		fmt.Println("[LOG] Буфер пуст, нечего извлекать")
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	fmt.Printf("[LOG] Извлечение данных из буфера \n")
	var output []int = r.array[:r.pos+1]
	// Виртуальная очистка нашего буфера
	r.pos = -1
	return output
}

// 1. Стадия фильтрации отрицательных чисел (не пропускать отрицательные числа).
func filtrNegatives(done <-chan interface{}, input <-chan int) <-chan int {
	fmt.Println("[LOG] Запуск стадии фильтрации отрицательных чисел")
	multipliedStream := make(chan int)
	go func() {
		defer close(multipliedStream)
		for {
			select {
			case <-done:
				fmt.Println("[LOG] Фильтр отрицательных чисел: получен сигнал завершения")
				return
			case v, isChannelOpen := <-input:
				if !isChannelOpen {
					fmt.Println("[LOG] Фильтр отрицательных чисел: входной канал закрыт")
					return
				}
				if v >= 0 {
					fmt.Printf("[LOG] Фильтр отрицательных чисел: число %d прошло фильтр\n", v)
					select {
					case multipliedStream <- v:
					case <-done:
						return
					}
				} else {
					fmt.Printf("[LOG] Фильтр отрицательных чисел: число %d отфильтровано (отрицательное)\n", v)
					fmt.Printf("Число %d не прошло фильтр на отрицательные числа!\n", v)
				}
			}
		}
	}()
	return multipliedStream
}

// 2. Стадия фильтрации чисел, не кратных 3 (не пропускать такие числа), исключая также и 0.
func filtrThree(done <-chan interface{}, input <-chan int) <-chan int {
	fmt.Println("[LOG] Запуск стадии фильтрации чисел, не кратных 3")
	multipliedStream := make(chan int)
	go func() {
		defer close(multipliedStream)
		for {
			select {
			case <-done:
				fmt.Println("[LOG] Фильтр кратных 3: получен сигнал завершения")
				return
			case v, isChannelOpen := <-input:
				if !isChannelOpen {
					fmt.Println("[LOG] Фильтр кратных 3: входной канал закрыт")
					return
				}

				if v != 0 {
					if v%3 == 0 {
						fmt.Printf("[LOG] Фильтр кратных 3: число %d прошло фильтр\n", v)
						select {
						case multipliedStream <- v:
						case <-done:
							return
						}
					} else {
						fmt.Printf("[LOG] Фильтр кратных 3: число %d отфильтровано (не кратно 3)\n", v)
						fmt.Printf("Число %d не кратно 3 - не прошло фильтр!\n", v)
					}
				} else {
					fmt.Printf("[LOG] Фильтр кратных 3: число %d отфильтровано (равно 0)\n", v)
					fmt.Printf("Число %d не прошло фильтр на 0!\n", v)
				}
			}
		}
	}()
	return multipliedStream
}

// 3. Стадия буферизации данных в кольцевом буфере с интерфейсом, соответствующим тому, который
// был дан в качестве задания в 19 модуле. В этой стадии предусмотреть опустошение буфера (и соответственно,
// передачу этих данных, если они есть, дальше) с определённым интервалом во времени. Значения размера буфера
// и этого интервала времени сделать настраиваемыми (как мы делали: через константы или глобальные переменные).
func bufferisation(done <-chan interface{}, c <-chan int) <-chan int {
	fmt.Printf("[LOG] Запуск стадии буферизации (размер буфера: %d, интервал очистки: %v)\n", bufferSize, bufferDrainInterval)
	bufferedIntChan := make(chan int)
	buffer := NewRingIntBuffer(bufferSize)
	go func() {
		defer fmt.Println("[LOG] Буферизация: завершение горутины приема данных")
		for {
			select {
			case data := <-c:
				fmt.Printf("[LOG] Буферизация: получено число %d\n", data)
				buffer.Push(data)
			case <-done:
				return
			}
		}
	}()

	// просмотр буфера с заданным интервалом времени - bufferDrainInterval
	go func() {
		defer fmt.Println("[LOG] Буферизация: завершение горутины очистки буфера")
		for {
			select {
			case <-time.After(bufferDrainInterval):
				fmt.Println("[LOG] Буферизация: истечение интервала очистки буфера")
				bufferData := buffer.Get()
				// Если в кольцевом буфере что-то есть - выводим содержимое построчно
				if bufferData != nil {
					fmt.Printf("[LOG] Буферизация: отправка %d элементов в следующий этап\n", len(bufferData))
					fmt.Print("Получены данные: ")
					for _, data := range bufferData {
						select {
						case bufferedIntChan <- data:
							fmt.Printf("[LOG] Буферизация: отправлено число %d\n", data)
						case <-done:
							return
						}
					}
				}
				fmt.Println()
			case <-done:
				return
			}
		}
	}()
	return bufferedIntChan
}

// источник данных для конвейера. Непосредственным источником данных должна быть консоль.
func startDataSource() (chan interface{}, chan int) {
	fmt.Println("[LOG] Инициализация источника данных")

	var sinp string                    // хранит считанную строку
	var isinp int                      // хранит сконвертированное число
	var err error                      // срабатывает при ошибке конвертации
	var ints = make(chan int)          // канал полученных данных
	var ended = make(chan interface{}) // флаг окончания для выхода из программы

	fmt.Println("Введите данные (для завершения просто нажмите \"Ввод\"):")
	go func() {
		defer close(ints)
		defer close(ended)
		for {
			fmt.Scanf("%s\n", &sinp)
			isinp, err = strconv.Atoi(sinp)
			if sinp == "" {
				fmt.Println("[LOG] Источник данных: получена пустая строка - завершение работы")
				fmt.Printf("Выход")
				return
			}
			if err == nil {
				fmt.Printf("[LOG] Источник данных: успешно преобразовано число")
				ints <- isinp
			} else {
				fmt.Printf("[LOG] Источник данных: ошибка преобразования")
				fmt.Printf("Необходимо вводить только целые числа размером 32 бит! Ошибка: %v\n", err)
			}
			sinp = ""
		}
	}()
	return ended, ints
}

func main() {
	fmt.Println("[LOG] Запуск программы")
	// источник данных для конвейера
	ended, intStream := startDataSource()

	// 1. Стадия фильтрации отрицательных чисел (не пропускать отрицательные числа).
	// 2. Стадия фильтрации чисел, не кратных 3 (не пропускать такие числа), исключая также и 0.
	// 3. Буфферизация
	pipe := bufferisation(ended, filtrThree(ended, filtrNegatives(ended, intStream)))

	// Потребитель данных конвейера. Данные от конвейера можно направить снова в консоль построчно:
	go func() {
		fmt.Println("[LOG] Запуск потребителя данных конвейера")
		for result := range pipe {
			fmt.Printf("Обработанные данные: %v\n", result)
			fmt.Printf("%v ", result)
		}
		fmt.Println("[LOG] Потребитель данных: канал закрыт")
	}()

	// и одна горутина для ожидания выхода из программы
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		fmt.Println("[LOG] Запуск горутины ожидания завершения")
		select {
		case <-ended: // завершение при закрытии канала
		        fmt.Println("[LOG] Получен сигнал завершения работы")
			wg.Done()
			return
		}
	}()
	wg.Wait()
	fmt.Println("[LOG] Программа завершена")
}

