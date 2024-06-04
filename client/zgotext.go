// Code generated by running "go generate" in golang.org/x/text. DO NOT EDIT.

package client

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type dictionary struct {
	index []uint32
	data  string
}

func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p, ok := messageKeyToIndex[key]
	if !ok {
		return "", false
	}
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}

func init() {
	dict := map[string]catalog.Dictionary{
		"da": &dictionary{index: daIndex, data: daData},
		"de": &dictionary{index: deIndex, data: deData},
		"en": &dictionary{index: enIndex, data: enData},
		"es": &dictionary{index: esIndex, data: esData},
		"fr": &dictionary{index: frIndex, data: frData},
		"it": &dictionary{index: itIndex, data: itData},
		"nl": &dictionary{index: nlIndex, data: nlData},
		"sl": &dictionary{index: slIndex, data: slData},
		"uk": &dictionary{index: ukIndex, data: ukData},
	}
	fallback := language.MustParse("en")
	cat, err := catalog.NewFromMap(dict, catalog.Fallback(fallback))
	if err != nil {
		panic(err)
	}
	message.DefaultCatalog = cat
}

var messageKeyToIndex = map[string]int{
	"%s. The cause of the error is: %s.":                                                            12,
	"An internal error occurred":                                                                    2,
	"Failed to add a secure internet server with organisation ID: '%s'":                             4,
	"Failed to add a server with URL: '%s'":                                                         5,
	"Failed to add an institute access server with URL: '%s'":                                       3,
	"Failed to connect to server: '%s'":                                                             7,
	"Failed to obtain a VPN configuration for server: '%s'":                                         8,
	"Failed to obtain the list of organizations":                                                    9,
	"Failed to obtain the list of servers":                                                          10,
	"The client tried to autoconnect to the VPN server: '%s', but the operation failed to complete": 1,
	"The client tried to autoconnect to the VPN server: '%s', but you need to authorizate again. Please manually connect again.": 0,
	"The input: '%s' is not a valid URL":   6,
	"Timeout reached contacting URL: '%s'": 11,
}

var daIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const daData string = ""

var deIndex = []uint32{ // 14 elements
	0x00000000, 0x000000a7, 0x0000012d, 0x00000151,
	0x00000151, 0x00000151, 0x00000151, 0x00000151,
	0x00000151, 0x00000151, 0x00000151, 0x00000151,
	0x00000151, 0x00000151,
} // Size: 80 bytes

const deData string = "" + // Size: 337 bytes
	"\x02Der Client hat versucht, sich automatisch mit dem VPN-Server '%[1]s'" +
	" zu verbinden, aber Sie müssen sich erneut autorisieren. Bitte verbinden" +
	" sie sich manuell erneut.\x02Der Client hat versucht, sich automatisch m" +
	"it dem VPN-Server '%[1]s' zu verbinden, aber der Vorgang konnte nicht ab" +
	"geschlossen werden\x02Ein interner Fehler ist aufgetreten"

var enIndex = []uint32{ // 14 elements
	0x00000000, 0x0000007e, 0x000000df, 0x000000fa,
	0x00000135, 0x0000017a, 0x000001a3, 0x000001c9,
	0x000001ee, 0x00000227, 0x00000252, 0x00000277,
	0x0000029f, 0x000002c8,
} // Size: 80 bytes

const enData string = "" + // Size: 712 bytes
	"\x02The client tried to autoconnect to the VPN server: '%[1]s', but you " +
	"need to authorizate again. Please manually connect again.\x02The client " +
	"tried to autoconnect to the VPN server: '%[1]s', but the operation faile" +
	"d to complete\x02An internal error occurred\x02Failed to add an institut" +
	"e access server with URL: '%[1]s'\x02Failed to add a secure internet ser" +
	"ver with organisation ID: '%[1]s'\x02Failed to add a server with URL: '%" +
	"[1]s'\x02The input: '%[1]s' is not a valid URL\x02Failed to connect to s" +
	"erver: '%[1]s'\x02Failed to obtain a VPN configuration for server: '%[1]" +
	"s'\x02Failed to obtain the list of organizations\x02Failed to obtain the" +
	" list of servers\x02Timeout reached contacting URL: '%[1]s'\x02%[1]s. Th" +
	"e cause of the error is: %[2]s."

var esIndex = []uint32{ // 14 elements
	0x00000000, 0x0000008d, 0x000000f0, 0x00000109,
	0x00000109, 0x00000109, 0x00000109, 0x00000109,
	0x00000109, 0x00000109, 0x00000109, 0x00000109,
	0x00000109, 0x00000109,
} // Size: 80 bytes

const esData string = "" + // Size: 265 bytes
	"\x02El cliente intentó autoconectarse al servidor VPN: '%[1]s', pero nec" +
	"esita autorizarse de nuevo. Por favor, conéctese manualmente de nuevo." +
	"\x02El cliente intentó autoconectarse al servidor VPN: %[1]s', pero la o" +
	"peración no se ha completado\x02Se ha producido un error"

var frIndex = []uint32{ // 14 elements
	0x00000000, 0x000000a0, 0x0000010e, 0x00000130,
	0x0000017e, 0x000001da, 0x00000210, 0x0000023e,
	0x0000026c, 0x000002b3, 0x000002e4, 0x00000310,
	0x00000310, 0x00000310,
} // Size: 80 bytes

const frData string = "" + // Size: 784 bytes
	"\x02Le client a essayé de se connecter automatiquement au serveur VPN" +
	"\u202f: '%[1]s', mais vous devez l'autoriser de nouveau. Veuillez vous r" +
	"econnecter manuellement.\x02Le client a essayé de se connecter automatiq" +
	"uement au serveur VPN\u202f: '%[1]s', mais l'opération a échouée\x02Une " +
	"erreur interne s'est produite\x02Échec de l'ajout d'un serveur d'accès à" +
	" un institut avec l'URL\u202f: '%[1]s'\x02Échec de l'ajout d'un serveur " +
	"d'accès à un institut avec l'ID d'organisation\u202f: '%[1]s'\x02Échec d" +
	"e l'ajout d'un serveur avec l'URL\u202f: '%[1]s'\x02L'entrée\u202f: '%[1" +
	"]s' n'est pas un URL valide\x02Échec de la connexion au serveur\u202f: '" +
	"%[1]s'\x02Échec d'obtention d'une configuration VPN pour le serveur" +
	"\u202f: '%[1]s'\x02Échec de l'obtention de liste des organisations\x02Éc" +
	"hec l'obtention de la liste des serveurs"

var itIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const itData string = ""

var nlIndex = []uint32{ // 14 elements
	0x00000000, 0x00000088, 0x000000df, 0x000000fe,
	0x0000014b, 0x000001a2, 0x000001de, 0x0000020e,
	0x00000241, 0x00000289, 0x000002c1, 0x000002f4,
	0x00000324, 0x0000034f,
} // Size: 80 bytes

const nlData string = "" + // Size: 847 bytes
	"\x02De client wilde automatisch verbinden met de VPN server: '%[1]s', ma" +
	"ar er is geen geldige authorizatie. Verbind handmatig nog een keer.\x02D" +
	"e client wilde automatisch verbinden met de VPN server: '%[1]s', maar he" +
	"t was mislukt\x02Een interne fout is opgetreden\x02Het is mislukt om een" +
	" institute access server toe te voegen met URL: '%[1]s'\x02Het is misluk" +
	"t om een secure internet server toe te voegen met organisatie ID: '%[1]s" +
	"'\x02Het is mislukt om een server toe te voegen met URL: '%[1]s'\x02Het " +
	"ingegeven veld: '%[1]s' is geen geldige URL\x02Het is mislukt om te verb" +
	"inden met server: '%[1]s'\x02Het is mislukt om een VPN configuratie op t" +
	"e halen voor server: '%[1]s'\x02Het is mislukt om de lijst van organisat" +
	"ies op te halen\x02Het is mislukt om de lijst van servers op te halen" +
	"\x02Er is een time-out opgetreden voor URL: '%[1]s'\x02%[1]s. The oorzaa" +
	"k van de error is: %[2]s."

var slIndex = []uint32{ // 14 elements
	0x00000000, 0x0000008b, 0x000000e6, 0x00000104,
	0x00000154, 0x0000019d, 0x000001d0, 0x000001ed,
	0x0000021a, 0x00000256, 0x00000282, 0x000002ae,
	0x000002f4, 0x00000313,
} // Size: 80 bytes

const slData string = "" + // Size: 787 bytes
	"\x02Odjemalec se je poskusil samodejno povezati s strežnikom VPN \x22%[1" +
	"]s\x22, vendar ga morate ponovno avtorizirati. Ponovno se povežite ročno" +
	".\x02Odjemalec se je poskusil samodejno povezati s strežnikom VPN \x22%[" +
	"1]s\x22, vendar mu ni uspelo\x02Prišlo je do notranje napake\x02Napaka p" +
	"ri dodajanju strežnika za dostop do ustanove. Strežnikov URL: '%[1]s'" +
	"\x02Napaka pri dodajanju strežnika za varni splet. Strežnikov URL: '%[1]" +
	"s'\x02Napaka pri dodajanju strežnika z URL-jem: '%[1]s'\x02Vnos \x22%[1]" +
	"s\x22 ni veljaven URL\x02Napaka pri povezovanju s strežnikom \x22%[1]s" +
	"\x22\x02Napaka pri pridobivanju nastavitve VPN za strežnik \x22%[1]s\x22" +
	"\x02Napaka pri pridobivanju seznama organizacij\x02Napaka pri pridobivan" +
	"ju seznama strežnikov\x02Pri dostopu do URL-ja \x22%[1]s\x22 je prišlo d" +
	"o preteka časovne kontrole\x02%[1]s. Vzrok napake je: %[2]s."

var ukIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const ukData string = ""

// Total table size 4452 bytes (4KiB); checksum: EADB3284
