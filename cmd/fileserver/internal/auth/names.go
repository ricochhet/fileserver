package auth

import "hash/fnv"

var nameAdjectives = []string{
	"Amber", "Azure", "Brass", "Bright", "Calm", "Clever", "Cobalt",
	"Coral", "Crisp", "Dark", "Dawn", "Deep", "Dusty", "Early",
	"Ember", "Faint", "Firm", "Fleet", "Frost", "Gentle", "Gilt",
	"Gold", "Grand", "Gray", "Green", "Hazy", "High", "Iron",
	"Jade", "Keen", "Kind", "Late", "Light", "Lone", "Lucky",
	"Mild", "Misty", "Muted", "Navy", "Noble", "Pale", "Pine",
	"Prime", "Quiet", "Rapid", "Red", "Rich", "Rosy", "Royal",
	"Rusty", "Sand", "Sharp", "Shy", "Silver", "Slim", "Soft",
	"Stern", "Still", "Stone", "Storm", "Strong", "Sunny", "Swift",
	"Tall", "Teal", "Thin", "True", "Warm", "Wild", "Wise",
}

var nameAnimals = []string{
	"Albatross", "Axolotl", "Badger", "Bear", "Beaver", "Bison",
	"Bobcat", "Buffalo", "Camel", "Capybara", "Caribou", "Cheetah",
	"Chinchilla", "Condor", "Coyote", "Crane", "Crow", "Dingo",
	"Dolphin", "Donkey", "Dove", "Eagle", "Egret", "Elk", "Falcon",
	"Ferret", "Finch", "Fox", "Gecko", "Gopher", "Grouse", "Hare",
	"Hawk", "Heron", "Ibis", "Jackal", "Jaguar", "Jay", "Kestrel",
	"Kite", "Lemur", "Leopard", "Loon", "Lynx", "Marten", "Mink",
	"Mole", "Moose", "Narwhal", "Newt", "Ocelot", "Osprey", "Otter",
	"Owl", "Panther", "Pelican", "Pika", "Porcupine", "Puffin",
	"Quail", "Raven", "Robin", "Salamander", "Seal", "Shrew",
	"Skunk", "Sloth", "Sparrow", "Stoat", "Swift", "Tapir",
	"Thrush", "Tiger", "Vole", "Vulture", "Walrus", "Weasel",
	"Wolf", "Wolverine", "Wombat", "Wren", "Yak",
}

// GenerateDisplayName derives a stable adjective-animal display name from a
// username using FNV-32a so the same username always produces the same name.
func GenerateDisplayName(username string) string {
	h := fnv.New32a()
	h.Write([]byte(username))
	n := h.Sum32()

	return nameAdjectives[n%uint32(len(nameAdjectives))] +
		nameAnimals[(n>>8)%uint32(len(nameAnimals))]
}
