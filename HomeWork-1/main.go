package main

import (
	"strings"
)

/*
	код писать в этом файле
	наверняка у вас будут какие-то структуры с методами, глобальные переменные ( тут можно ), функции
*/

type Item struct{
	Name string
	Location string
}

type Room struct{
	Name string
	EnterMsg string
	Exits	[]string
	ExistMap map[string]string
	Items	[]*Item
	LookFunc func() string
}

type Player struct{
	Room *Room
	Inventary map[string]bool
	HasBackpack bool
}

var player *Player
var world map[string]*Room
var doorOpen bool

func main() {
	/*
		в этой функции можно ничего не писать,
		но тогда у вас не будет работать через go run main.go
		очень круто будет сделать построчный ввод команд тут, хотя это и не требуется по заданию
	*/

	initGame()

	_ = handleCommand("some command")
}

func initGame() {
	doorOpen = false
	player = &Player{
				Inventary: make(map[string]bool),
	} 
	world := make(map[string]*Room)

	kitchen := &Room{
		Name: "Кухня",
		EnterMsg: "кухня, ничего интересного. можно пройти - коридор",
		Exits: []string{"коридор"},
		ExistMap: map[string]string{"коридор": "коридор"},
		Items: []*Item{{Name:"чай", Location: "стол"}},
	}

	kitchen.LookFunc = func() string {
		goal := "надо собрать рюкзак и идти в универ"
		if player.HasBackpack {
			goal = "надо идти в универ"
		}

		return "ты находишься на кухне, на столе: чай, " + goal + ". можно пройти - коридор"
	}

	world["кухня"] = kitchen

	corridor := &Room{
		Name: "коридор",
		EnterMsg: "ничего интересного. можно пройти - кухня, комната, улица",
		Exits: []string{"кухня", "комната", "улица"},
		ExistMap: map[string]string{"кухня": "кухня", 
									"комната": "комната", 
									"улица": "улица"},
	}

	corridor.LookFunc = func() string{
		return corridor.EnterMsg
	} 

	world["коридор"] = corridor

	room := &Room{
		Name: "комната",
		EnterMsg: "ты в своей комнате. можно пройти - коридор",
		Exits: []string{"коридор"},
		ExistMap: map[string]string{"коридор": "коридор"},
		Items: []*Item{
			{Name: "ключи", Location: "стол"},
			{Name: "конспекты", Location: "стол"},
			{Name: "рюкзвк", Location: "стул"},
		},
	}
	room.LookFunc = func() string{
		if len(room.Items) == 0{
			return "пустая комната. можно пройти - коридор"
		}

		tableItems := []string{}
		chairItems := []string{}

		for _, item := range room.Items{
			if item.Location == "стол"{
				tableItems = append(tableItems, item.Name)
			} else if item.Location == "стул"{
				chairItems = append(chairItems, item.Name)
			}
		}

		parts := []string{}
		if len(tableItems) > 0 {
			parts = append(parts, "на столе: "+strings.Join(tableItems, ", "))
		}
		if len(chairItems) > 0 {
			parts = append(parts, "на стуле: "+strings.Join(chairItems, ", "))
		}

		return strings.Join(parts, ", ") + ". можно пройти - коридор"
	}

	world["комната"] = room

	street := &Room{
		Name: "улица",
		EnterMsg: "на улице весна. можно пройти - домой",
		Exits: []string{"домой"},
		ExistMap: map[string]string{"домой": "коридор"},
	}

	street.LookFunc = func() string{
		return street.EnterMsg
	}

	world["улица"] = street

	player.Room = kitchen
}

func handleCommand(command string) string {
	parts := strings.Split(command, "")
	if len(parts) == 0{
		return "неизвестная команда"
	}

	switch parts[0]{
	case "осмотреться":
		return player.Room.LookFunc()
	case "идти":
		if len(parts) < 2{
			return "неизвестная команда"
		}
		dest := parts[1]
		roomName, ok := player.Room.ExistMap[dest]
		if !ok {
			return "нет пути в " + dest
		} 
		if dest == "улица" && !doorOpen{
			return "дверь закрыта"
		}
		player.Room = world[roomName]
		return player.Room.EnterMsg
	case "взять":
		if len(parts) < 2{
			return "неизвестная команда"
		}

		itemName := parts[1]
		if !player.HasBackpack {
			return "некуда класть"
		}

		for i, item := range player.Room.Items{
			if item.Name == itemName{
				player.Room.Items = append(player.Room.Items[:i], player.Room.Items[i+1:]...)
				player.Inventary[itemName] = true

				return "предмет добавлен в инвентарь: " + itemName
			}
		}
		return "нет такого"

	case "надеть":
		if len(parts) < 2 {
			return "неизвестная команда"
		}
		itemName := parts[1]
		if itemName == "рюкзак"{
			for i, item := range player.Room.Items {
				if item.Name == "рюкзак"{
					player.Room.Items = append(player.Room.Items[:i], player.Room.Items[i+1:]...)
					player.HasBackpack = true
					return "вы надели: рюкзак"
				}
			}
		}
		return "нет такого"

	case "применить":
		if len(parts) < 3{
			return "неизвестная команда"
		}

		itemName := parts[1]
		target := parts[2]

		if !player.Inventary[itemName]{
			return "нет предмета в инвентаре - " + itemName
		}

		if itemName == "ключи" && target == "дверь" && player.Room.Name == "коридор"{
			doorOpen = true
			return "дверь открыта"
		}

		return "не к чему применить"
	
	default:
		return "неизвестная команда"
	}
}
