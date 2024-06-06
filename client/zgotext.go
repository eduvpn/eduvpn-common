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
		"pt": &dictionary{index: ptIndex, data: ptData},
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
	"%s. The cause of the error is: %s.":                                                            11,
	"An internal error occurred":                                                                    12,
	"Failed to add a secure internet server with organisation ID: '%s'":                             1,
	"Failed to add a server with URL: '%s'":                                                         2,
	"Failed to add an institute access server with URL: '%s'":                                       0,
	"Failed to connect to server: '%s'":                                                             6,
	"Failed to obtain a VPN configuration for server: '%s'":                                         7,
	"Failed to obtain the list of organizations":                                                    8,
	"Failed to obtain the list of servers":                                                          9,
	"The client tried to autoconnect to the VPN server: '%s', but the operation failed to complete": 5,
	"The client tried to autoconnect to the VPN server: '%s', but you need to authorizate again. Please manually connect again.": 4,
	"The input: '%s' is not a valid URL":   3,
	"Timeout reached contacting URL: '%s'": 10,
}

var daIndex = []uint32{ // 14 elements
	0x00000000, 0x00000046, 0x0000009b, 0x000000ce,
	0x000000ff, 0x00000187, 0x000001e4, 0x0000020d,
	0x00000244, 0x00000274, 0x0000029b, 0x000002ce,
	0x000002ec, 0x00000305,
} // Size: 80 bytes

const daData string = "" + // Size: 773 bytes
	"\x02Kunne ikke tilføje en server for institutadgang med URL’en '%[1]s'" +
	"\x02Kunne ikke tilføje en server for sikkert internet med organisations-" +
	"ID’et '%[1]s'\x02Kunne ikke tilføje en server med URL’en '%[1]s'\x02Inpu" +
	"ttet '%[1]s' er altså ikke nogen gyldig URL\x02Klienten forsøgte at forb" +
	"inde automatisk til VPN-serveren '%[1]s, men dét kræver din fornyede god" +
	"kendelse. Forbind venligst manuelt.\x02Klienten forsøgte at forbinde til" +
	" VPN-serveren '%[1]s', men forsøget kunne ikke fuldføres\x02Kunne ikke f" +
	"orbinde til serveren '%[1]s'\x02Kunne ikke få en VPN-konfiguration for s" +
	"erven '%[1]s'\x02Kunne ikke få fat i listen over organisationer\x02Kunne" +
	" ikke få fat i listen af servere\x02Timeout i forsøget på at tilgå URL’e" +
	"n '%[1]s'\x02%[1]s. Fejlen skyldes: %[2]s.\x02Der skete en intern fejl"

var deIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x000000a7, 0x0000012d, 0x0000012d,
	0x0000012d, 0x0000012d, 0x0000012d, 0x0000012d,
	0x0000012d, 0x00000151,
} // Size: 80 bytes

const deData string = "" + // Size: 337 bytes
	"\x02Der Client hat versucht, sich automatisch mit dem VPN-Server '%[1]s'" +
	" zu verbinden, aber Sie müssen sich erneut autorisieren. Bitte verbinden" +
	" sie sich manuell erneut.\x02Der Client hat versucht, sich automatisch m" +
	"it dem VPN-Server '%[1]s' zu verbinden, aber der Vorgang konnte nicht ab" +
	"geschlossen werden\x02Ein interner Fehler ist aufgetreten"

var enIndex = []uint32{ // 14 elements
	0x00000000, 0x0000003b, 0x00000080, 0x000000a9,
	0x000000cf, 0x0000014d, 0x000001ae, 0x000001d3,
	0x0000020c, 0x00000237, 0x0000025c, 0x00000284,
	0x000002ad, 0x000002c8,
} // Size: 80 bytes

const enData string = "" + // Size: 712 bytes
	"\x02Failed to add an institute access server with URL: '%[1]s'\x02Failed" +
	" to add a secure internet server with organisation ID: '%[1]s'\x02Failed" +
	" to add a server with URL: '%[1]s'\x02The input: '%[1]s' is not a valid " +
	"URL\x02The client tried to autoconnect to the VPN server: '%[1]s', but y" +
	"ou need to authorizate again. Please manually connect again.\x02The clie" +
	"nt tried to autoconnect to the VPN server: '%[1]s', but the operation fa" +
	"iled to complete\x02Failed to connect to server: '%[1]s'\x02Failed to ob" +
	"tain a VPN configuration for server: '%[1]s'\x02Failed to obtain the lis" +
	"t of organizations\x02Failed to obtain the list of servers\x02Timeout re" +
	"ached contacting URL: '%[1]s'\x02%[1]s. The cause of the error is: %[2]s" +
	".\x02An internal error occurred"

var esIndex = []uint32{ // 14 elements
	0x00000000, 0x00000054, 0x000000a7, 0x000000d7,
	0x00000101, 0x0000018e, 0x000001f1, 0x0000021c,
	0x0000025e, 0x00000295, 0x000002c1, 0x00000305,
	0x0000032a, 0x00000343,
} // Size: 80 bytes

const esData string = "" + // Size: 835 bytes
	"\x02Error al agregar el servidor de acceso a la institución. URL del ser" +
	"vidor: '%[1]s'\x02No se pudo añadir un servidor de internet seguro con I" +
	"D de organización: '%[1]s'\x02No se pudo añadir un servidor con URL: '%[" +
	"1]s'\x02La entrada: '%[1]s' no es una URL válida\x02El cliente intentó a" +
	"utoconectarse al servidor VPN: '%[1]s', pero necesita autorizarse de nue" +
	"vo. Por favor, conéctese manualmente de nuevo.\x02El cliente intentó aut" +
	"oconectarse al servidor VPN: %[1]s', pero la operación no se ha completa" +
	"do\x02Error al conectar con el servidor: '%[1]s'\x02Error al obtener una" +
	" configuración VPN para el servidor: '%[1]s'\x02No se ha podido obtener " +
	"la lista de las organizaciones\x02Error al obtener la lista de los servi" +
	"dores\x02Se ha alcanzado el tiempo de espera para conectar con la URL: %" +
	"[1]s\x02%[1]s. La causa del error es: %[2]s.\x02Se ha producido un error"

var frIndex = []uint32{ // 14 elements
	0x00000000, 0x0000004e, 0x000000aa, 0x000000e0,
	0x0000010e, 0x000001ae, 0x0000021c, 0x0000024a,
	0x00000291, 0x000002c2, 0x000002ee, 0x00000324,
	0x00000353, 0x00000375,
} // Size: 80 bytes

const frData string = "" + // Size: 885 bytes
	"\x02Échec de l'ajout d'un serveur d'accès à un institut avec l'URL\u202f" +
	": '%[1]s'\x02Échec de l'ajout d'un serveur d'accès à un institut avec l'" +
	"ID d'organisation\u202f: '%[1]s'\x02Échec de l'ajout d'un serveur avec l" +
	"'URL\u202f: '%[1]s'\x02L'entrée\u202f: '%[1]s' n'est pas un URL valide" +
	"\x02Le client a essayé de se connecter automatiquement au serveur VPN" +
	"\u202f: '%[1]s', mais vous devez l'autoriser de nouveau. Veuillez vous r" +
	"econnecter manuellement.\x02Le client a essayé de se connecter automatiq" +
	"uement au serveur VPN\u202f: '%[1]s', mais l'opération a échouée\x02Éche" +
	"c de la connexion au serveur\u202f: '%[1]s'\x02Échec d'obtention d'une c" +
	"onfiguration VPN pour le serveur\u202f: '%[1]s'\x02Échec de l'obtention " +
	"de liste des organisations\x02Échec l'obtention de la liste des serveurs" +
	"\x02Délai maximal atteint pour contacter l'URL\u202f: %[1]s\x02%[1]s. La" +
	" cause de cette erreur est\u202f: %[2]s.\x02Une erreur interne s'est pro" +
	"duite"

var itIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const itData string = ""

var nlIndex = []uint32{ // 14 elements
	0x00000000, 0x0000004d, 0x000000a4, 0x000000e0,
	0x00000110, 0x00000198, 0x000001ef, 0x00000222,
	0x0000026a, 0x000002a2, 0x000002d5, 0x00000305,
	0x00000330, 0x0000034f,
} // Size: 80 bytes

const nlData string = "" + // Size: 847 bytes
	"\x02Het is mislukt om een institute access server toe te voegen met URL:" +
	" '%[1]s'\x02Het is mislukt om een secure internet server toe te voegen m" +
	"et organisatie ID: '%[1]s'\x02Het is mislukt om een server toe te voegen" +
	" met URL: '%[1]s'\x02Het ingegeven veld: '%[1]s' is geen geldige URL\x02" +
	"De client wilde automatisch verbinden met de VPN server: '%[1]s', maar e" +
	"r is geen geldige authorizatie. Verbind handmatig nog een keer.\x02De cl" +
	"ient wilde automatisch verbinden met de VPN server: '%[1]s', maar het wa" +
	"s mislukt\x02Het is mislukt om te verbinden met server: '%[1]s'\x02Het i" +
	"s mislukt om een VPN configuratie op te halen voor server: '%[1]s'\x02He" +
	"t is mislukt om de lijst van organisaties op te halen\x02Het is mislukt " +
	"om de lijst van servers op te halen\x02Er is een time-out opgetreden voo" +
	"r URL: '%[1]s'\x02%[1]s. The oorzaak van de error is: %[2]s.\x02Een inte" +
	"rne fout is opgetreden"

var ptIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const ptData string = ""

var slIndex = []uint32{ // 14 elements
	0x00000000, 0x00000050, 0x00000099, 0x000000cc,
	0x000000e9, 0x00000174, 0x000001cf, 0x000001fc,
	0x00000238, 0x00000264, 0x00000290, 0x000002d6,
	0x000002f5, 0x00000313,
} // Size: 80 bytes

const slData string = "" + // Size: 787 bytes
	"\x02Napaka pri dodajanju strežnika za dostop do ustanove. Strežnikov URL" +
	": '%[1]s'\x02Napaka pri dodajanju strežnika za varni splet. Strežnikov U" +
	"RL: '%[1]s'\x02Napaka pri dodajanju strežnika z URL-jem: '%[1]s'\x02Vnos" +
	" \x22%[1]s\x22 ni veljaven URL\x02Odjemalec se je poskusil samodejno pov" +
	"ezati s strežnikom VPN \x22%[1]s\x22, vendar ga morate ponovno avtorizir" +
	"ati. Ponovno se povežite ročno.\x02Odjemalec se je poskusil samodejno po" +
	"vezati s strežnikom VPN \x22%[1]s\x22, vendar mu ni uspelo\x02Napaka pri" +
	" povezovanju s strežnikom \x22%[1]s\x22\x02Napaka pri pridobivanju nasta" +
	"vitve VPN za strežnik \x22%[1]s\x22\x02Napaka pri pridobivanju seznama o" +
	"rganizacij\x02Napaka pri pridobivanju seznama strežnikov\x02Pri dostopu " +
	"do URL-ja \x22%[1]s\x22 je prišlo do preteka časovne kontrole\x02%[1]s. " +
	"Vzrok napake je: %[2]s.\x02Prišlo je do notranje napake"

var ukIndex = []uint32{ // 14 elements
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000,
} // Size: 80 bytes

const ukData string = ""

// Total table size 5976 bytes (5KiB); checksum: 4C67332F
