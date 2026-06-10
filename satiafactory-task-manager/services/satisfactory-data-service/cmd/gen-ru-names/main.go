// One-off generator: merges game-data item/recipe names with ru_names.json.
// Run: go run ./cmd/gen-ru-names -data ./data/game-data.json -out ./data/ru_names.json
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"
)

type greenyRoot struct {
	Items   map[string]greenyItem   `json:"items"`
	Recipes map[string]greenyRecipe `json:"recipes"`
}

type greenyItem struct {
	ClassName string `json:"className"`
	Name      string `json:"name"`
}

type greenyRecipe struct {
	ClassName string           `json:"className"`
	Name      string           `json:"name"`
	Products  []greenyItemAmt  `json:"products"`
}

type greenyItemAmt struct {
	Item string `json:"item"`
}

// Official Satisfactory Russian names (Update 1.0 + classics).
var enToRU = map[string]string{
	"Iron Plate": "Железная пластина", "Iron Ingot": "Железный слиток", "Iron Rod": "Железный стержень",
	"Iron Screw": "Винт", "Screw": "Винт", "Reinforced Iron Plate": "Армированная железная пластина",
	"Copper Ingot": "Медный слиток", "Copper Sheet": "Медный лист", "Cable": "Кабель", "Wire": "Провод",
	"Concrete": "Бетон", "Cement": "Цемент", "Steel Ingot": "Стальной слиток", "Steel Pipe": "Стальная труба",
	"Steel Beam": "Стальная балка", "Rotor": "Ротор", "Stator": "Статор", "Motor": "Двигатель",
	"Modular Frame": "Модульный каркас", "Heavy Modular Frame": "Тяжёлый модульный каркас",
	"Fused Modular Frame": "Слитый модульный каркас", "Encased Industrial Beam": "Закрытая стальная балка",
	"Circuit Board": "Печатная плата", "Computer": "Компьютер", "Supercomputer": "Суперкомпьютер",
	"Plastic": "Пластик", "Rubber": "Резина", "Quartz Crystal": "Кварц", "Silica": "Кремнезём",
	"Raw Quartz": "Необработанный кварц", "Aluminum Ingot": "Алюминиевый слиток",
	"Aluminum Casing": "Алюминиевый корпус", "Aluminum Scrap": "Алюминиевый лом",
	"Alclad Aluminum Sheet": "Алюминиевая пластина Alclad", "Aluminum Plate": "Алюминиевая пластина",
	"Coal": "Уголь", "Compacted Coal": "Спрессованный уголь", "Petroleum Coke": "Нефтяной кокс",
	"Iron Ore": "Железная руда", "Copper Ore": "Медная руда", "Limestone": "Известняк",
	"Caterium Ore": "Катерийная руда", "Caterium Ingot": "Катерийный слиток", "Sulfur": "Сера",
	"Bauxite": "Бокситы", "Water": "Вода", "Crude Oil": "Сырая нефть", "Oil": "Сырая нефть",
	"Fuel": "Топливо", "Liquid Biofuel": "Жидкое биотопливо", "Turbo Fuel": "Турботопливо",
	"Turbo Motor": "Турбомотор", "Heavy Oil Residue": "Мазут", "Polymer Resin": "Полимерная смола",
	"Fabric": "Ткань", "Biomass": "Биомасса", "Solid Biofuel": "Твёрдое биотопливо",
	"Nobelisk": "Нобелиск", "Gas Tank": "Газовый баллон", "Empty Canister": "Пустая канистра",
	"Packaged Fuel": "Топливо в канистре", "Packaged Oil": "Нефть в канистре",
	"Crystal Oscillator": "Кварцевый осциллятор", "High-Speed Connector": "Высокоскоростной соединитель",
	"Rocket Fuel": "Ракетное топливо", "Iodine Infused Filter": "Фильтр с йодом",
	"Hazmat Filter": "Фильтр для хазмат-костюма", "Battery": "Батарея", "Heat Sink": "Радиатор",
	"Cooling System": "Система охлаждения", "Ficsonium": "Фиксоний", "Ficsite Ingot": "Фикситовый слиток",
	"Ficsite Trigon": "Фикситовый тригон", "Dark Matter Crystal": "Кристалл тёмной материи",
	"Dark Matter Residue": "Остаток тёмной материи", "Time Crystal": "Кристалл времени",
	"SAM Fluctuator": "САМ-флуктуатор", "Reanimated SAM": "Реанимированный САМ",
	"Iron Powder": "Железный порошок", "SAM": "САМ", "SAM Ore": "САМ",
	"Uranium": "Уран", "Uranium Fuel Rod": "Урановый топливный стержень",
	"Encased Uranium Cell": "Закрытая урановая ячейка", "Non-Fissile Uranium": "Не-делимый уран",
	"Plutonium Pellet": "Плутониевый пеллет", "Plutonium Fuel Rod": "Плутониевый топливный стержень",
	"Plutonium Waste": "Плутониевые отходы", "Nuclear Waste": "Ядерные отходы",
	"Nitric Acid": "Азотная кислота", "Packaged Nitric Acid": "Азотная кислота в канистре",
	"Nitrogen Gas": "Азот", "Packaged Nitrogen Gas": "Азот в баллоне",
	"Alumina Solution": "Алюминиевый раствор", "Sulfuric Acid": "Серная кислота",
	"Packaged Sulfuric Acid": "Серная кислота в канистре",
	"Turbo Rifle Ammo": "Турбо-патроны",
	"Pressure Conversion Cube": "Куб преобразования давления", "Singularity Cell": "Сингулярная ячейка",
	"Ballistic Warp Drive": "Баллистический варп-двигатель", "AI Expansion Server": "Сервер расширения ИИ",
	"Nuclear Pasta": "Ядерная паста", "Magnetic Field Generator": "Генератор магнитного поля",
	"Thermal Propulsion Rocket": "Ракета термального движения", "Assembly Director System": "Система управления сборкой",
	"Biochemical Sculptor": "Биохимический скульптор", "Ficsonium Fuel Rod": "Фиксониевый топливный стержень",
	"Diamond": "Алмаз", "Excited Photonic Matter": "Возбуждённая фотонная материя",
	"Quantum Energy": "Квантовая энергия", "Superposition Oscillator": "Осциллятор суперпозиции",
	"Neural-Quantum Processor": "Нейро-квантовый процессор", "Dark Energy": "Тёмная энергия",
	"Alien Power Matrix": "Инопланетная силовая матрица", "Alien DNA Capsule": "Капсула инопланетной ДНК",
	"Alien Protein": "Инопланетный белок",
	"Portable Miner": "Портативный бур",
	"Ionized Fuel": "Ионизированное топливо", "Packaged Rocket Fuel": "Ракетное топливо в канистре",
	"Liquid Turbo Fuel": "Жидкое турботопливо", "Computer Super": "Суперкомпьютер",
	"Motor Lightweight": "Облегчённый двигатель", "Temporal Processor": "Временной процессор",
	"Quantum Oscillator": "Квантовый осциллятор", "Smart Plating": "Умная обшивка",
	"Versatile Framework": "Универсальный каркас", "Automated Wiring": "Автоматическая проводка",
	"Modular Engine": "Модульный двигатель", "Adaptive Control Unit": "Адаптивный блок управления",
	"Assembly System": "Сборочная система",
	"Uranium Ore": "Урановая руда", "Quartz": "Кварц",
	"Leaves": "Листья", "Wood": "Древесина", "Mycelia": "Мицелий", "Flower Petals": "Лепестки цветов",
	"Paleberry": "Бледноягода", "Bacon Agaric": "Бекон-агарик", "Beryl Nut": "Берилловый орех",
	"Power Shard": "Энергомодуль", "Power Slug": "Энергослизь", "Yellow Power Slug": "Жёлтая энергослизь",
	"Purple Power Slug": "Фиолетовая энергослизь", "Orange Power Slug": "Оранжевая энергослизь",
	"Conveyor Belt Mk.1": "Конвейерная лента Mk.1", "Conveyor Belt Mk.2": "Конвейерная лента Mk.2",
	"Conveyor Belt Mk.3": "Конвейерная лента Mk.3", "Conveyor Belt Mk.4": "Конвейерная лента Mk.4",
	"Conveyor Belt Mk.5": "Конвейерная лента Mk.5", "Conveyor Belt Mk.6": "Конвейерная лента Mk.6",
	"Pipeline Mk.1": "Труба Mk.1", "Pipeline Mk.2": "Труба Mk.2",
	"Assembler": "Сборочный цех", "Manufacturer": "Производитель", "Constructor": "Конструктор",
	"Smelter": "Плавильня", "Foundry": "Литейная", "Refinery": "Нефтеперерабатывающий завод",
	"Blender": "Смеситель", "Packager": "Упаковщик", "Converter": "Конвертер",
	"Particle Accelerator": "Ускоритель частиц", "Hadron Collider": "Адронный коллайдер",
	"Quantum Encoder": "Квантовый кодировщик", "Nuclear Power Plant": "Ядерный реактор",
	"Copper Dust": "Медная пыль",
	"Electromagnetic Control Rod": "Электромагнитный стержень управления",
	"Ficsite Ingot (Iron)": "Фикситовый слиток (железо)", "Ficsite Ingot (Aluminum)": "Фикситовый слиток (алюминий)",
	"Ficsite Ingot (Caterium)": "Фикситовый слиток (катерий)",
	"Wet Concrete": "Влажный бетон", "Wet Cement": "Влажный цемент",
	"Homing Rifle Ammo": "Самонаводящиеся патроны", "Rifle Ammo": "Патроны для винтовки",
	"Gas Mask": "Противогаз", "Hazmat Suit": "Хазмат-костюм", "Jetpack": "Реактивный ранец",
	"Hoverpack": "Ховерпак", "Blade Runners": "Клинобеги", "Nobelisk Detonator": "Детонатор нобелисков",
	"Object Scanner": "Сканер объектов", "Rifle": "Винтовка", "Xeno-Zapper": "Ксенозаппер",
	"Xeno-Basher": "Ксенобашер", "Zipline": "Зиплайн", "Chainsaw": "Бензопила",
	"High-Speed Circuit Board": "Высокоскоростная печатная плата", "Crystal": "Кристалл",
	"Dissolved Silica": "Растворённый кремнезём",
}

func main() {
	dataPath := flag.String("data", "./data/game-data.json", "game-data.json path")
	outPath := flag.String("out", "./data/ru_names.json", "output ru_names.json")
	existingPath := flag.String("merge", "", "optional existing ru_names to preserve")
	flag.Parse()

	raw, err := os.ReadFile(*dataPath)
	if err != nil {
		log.Fatal(err)
	}
	var root greenyRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		log.Fatal(err)
	}

	out := map[string]string{}
	if *existingPath != "" {
		mergeFile(out, *existingPath)
	} else if b, err := os.ReadFile(*outPath); err == nil {
		_ = json.Unmarshal(b, &out)
	}

	for class, item := range root.Items {
		if item.Name == "" {
			continue
		}
		if ru, ok := enToRU[item.Name]; ok {
			out[class] = ru
		} else if _, exists := out[class]; !exists {
			out[class] = item.Name
		}
	}

	for class, rec := range root.Recipes {
		if rec.Name == "" {
			continue
		}
		if ru, ok := enToRU[rec.Name]; ok {
			out[class] = ru
			continue
		}
		// Alternate recipe names: "Alternate: X" -> "Альтернатива: X"
		name := rec.Name
		if strings.HasPrefix(name, "Alternate: ") {
			inner := strings.TrimPrefix(name, "Alternate: ")
			if ru, ok := enToRU[inner]; ok {
				out[class] = "Альтернатива: " + ru
				continue
			}
		}
		if len(rec.Products) > 0 {
			prodClass := rec.Products[0].Item
			if ru := out[prodClass]; ru != "" && hasCyrillic(ru) {
				if strings.HasPrefix(name, "Alternate: ") {
					out[class] = "Альтернатива: " + ru
				} else {
					out[class] = ru
				}
				continue
			}
			if item, ok := root.Items[prodClass]; ok {
				if ru, ok := enToRU[item.Name]; ok {
					if strings.HasPrefix(name, "Alternate: ") {
						out[class] = "Альтернатива: " + ru
					} else {
						out[class] = ru
					}
					continue
				}
			}
		}
		if _, exists := out[class]; !exists {
			out[class] = rec.Name
		}
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]string, len(out))
	for _, k := range keys {
		ordered[k] = out[k]
	}

	b, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(*outPath, b, 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote %d names to %s", len(ordered), *outPath)
}

func hasCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

func mergeFile(dst map[string]string, path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &dst)
}
