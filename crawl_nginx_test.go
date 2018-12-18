package main

import (
	"github.com/terorie/od-database-crawler/fasturl"
	"testing"
)

func TestParseDirNginx(t *testing.T) {
	var u fasturl.URL
	err := u.Parse("https://the-eye.eu/public/")
	if err != nil {
		t.Fatal("Failed to parse URL", err)
	}

	links, err := ParseDir([]byte(nginxListing), &u)
	if err != nil {
		t.Fatal("Failed to extract links", err)
	}

	if len(links) != len(nginxLinks) {
		t.Fatalf("Expected %d links, got %d",
			len(nginxLinks), len(links))
	}

	for i := 0; i < len(links); i++ {
		gotLink := links[i].String()
		expLink := nginxLinks[i]

		if gotLink != expLink {
			t.Errorf(`Expected "%s" got "%s"`,
				expLink, gotLink)
		}
	}
}

var nginxLinks = []string {
	"https://the-eye.eu/public/AppleArchive/",
	"https://the-eye.eu/public/AudioBooks/",
	"https://the-eye.eu/public/Books/",
	"https://the-eye.eu/public/Comics/",
	"https://the-eye.eu/public/Games/",
	"https://the-eye.eu/public/Icons/",
	"https://the-eye.eu/public/Images/",
	"https://the-eye.eu/public/JFK_Files/",
	"https://the-eye.eu/public/MSDN/",
	"https://the-eye.eu/public/Music/",
	"https://the-eye.eu/public/Operating%20Systems/",
	"https://the-eye.eu/public/Posters/",
	"https://the-eye.eu/public/Psychedelics/",
	"https://the-eye.eu/public/Psychoactives/",
	"https://the-eye.eu/public/Radio/",
	"https://the-eye.eu/public/Random/",
	"https://the-eye.eu/public/Site-Dumps/",
	"https://the-eye.eu/public/Software/",
	"https://the-eye.eu/public/Strategic%20Intelligence%20Network/",
	"https://the-eye.eu/public/WorldTracker.org/",
	"https://the-eye.eu/public/concen.org/",
	"https://the-eye.eu/public/freenrg.info/",
	"https://the-eye.eu/public/murdercube.com/",
	"https://the-eye.eu/public/parazite/",
	"https://the-eye.eu/public/ripreddit/",
	"https://the-eye.eu/public/rom/",
	"https://the-eye.eu/public/touhou/",
	"https://the-eye.eu/public/vns/",
	"https://the-eye.eu/public/xbins/",
	"https://the-eye.eu/public/xbins.diodematrix/",
	"https://the-eye.eu/public/Rclone_for_Scrubs.pdf",
	"https://the-eye.eu/public/Wget_Linux_Guide.pdf",
	"https://the-eye.eu/public/Wget_Windows_Guide.pdf",
	"https://the-eye.eu/public/rclone_guide.pdf",
	"https://the-eye.eu/public/wget-noobs-guide.pdf",
	"https://the-eye.eu/public/xbox-scene_Aug2014.7z",
}

const nginxListing =
`<html>
<head><title>Index of /public/</title></head>
<body bgcolor="white">
<h1>Index of /public/</h1><hr><pre><a href="../">../</a>
<a href="AppleArchive/">AppleArchive/</a>                                      03-Nov-2017 18:13       -
<a href="AudioBooks/">AudioBooks/</a>                                        29-Sep-2018 19:47       -
<a href="Books/">Books/</a>                                             27-Nov-2018 17:50       -
<a href="Comics/">Comics/</a>                                            05-Nov-2018 21:37       -
<a href="Games/">Games/</a>                                             28-Nov-2018 11:54       -
<a href="Icons/">Icons/</a>                                             22-May-2018 07:47       -
<a href="Images/">Images/</a>                                            21-Jan-2018 03:21       -
<a href="JFK_Files/">JFK_Files/</a>                                         03-Nov-2017 17:03       -
<a href="MSDN/">MSDN/</a>                                              03-Nov-2017 15:48       -
<a href="Music/">Music/</a>                                             02-Mar-2018 15:47       -
<a href="Operating%20Systems/">Operating Systems/</a>                                 25-Apr-2018 07:18       -
<a href="Posters/">Posters/</a>                                           07-Jul-2018 01:12       -
<a href="Psychedelics/">Psychedelics/</a>                                      11-Apr-2018 05:45       -
<a href="Psychoactives/">Psychoactives/</a>                                     18-May-2018 02:58       -
<a href="Radio/">Radio/</a>                                             09-Jun-2018 15:49       -
<a href="Random/">Random/</a>                                            04-Dec-2018 12:33       -
<a href="Site-Dumps/">Site-Dumps/</a>                                        15-Dec-2018 11:04       -
<a href="Software/">Software/</a>                                          27-Nov-2017 00:22       -
<a href="Strategic%20Intelligence%20Network/">Strategic Intelligence Network/</a>                    17-Nov-2017 16:35       -
<a href="WorldTracker.org/">WorldTracker.org/</a>                                  12-Apr-2018 04:16       -
<a href="concen.org/">concen.org/</a>                                        08-Oct-2018 14:08       -
<a href="freenrg.info/">freenrg.info/</a>                                      19-Dec-2017 10:59       -
<a href="murdercube.com/">murdercube.com/</a>                                    06-Dec-2017 10:45       -
<a href="parazite/">parazite/</a>                                          20-Nov-2017 21:25       -
<a href="ripreddit/">ripreddit/</a>                                         04-Aug-2018 14:30       -
<a href="rom/">rom/</a>                                               28-Nov-2018 14:15       -
<a href="touhou/">touhou/</a>                                            03-Nov-2017 11:07       -
<a href="vns/">vns/</a>                                               03-Nov-2017 11:36       -
<a href="xbins/">xbins/</a>                                             03-Nov-2017 17:23       -
<a href="xbins.diodematrix/">xbins.diodematrix/</a>                                 21-Sep-2018 22:33       -
<a href="Rclone_for_Scrubs.pdf">Rclone_for_Scrubs.pdf</a>                              04-Sep-2018 13:31    315K
<a href="Wget_Linux_Guide.pdf">Wget_Linux_Guide.pdf</a>                               21-Dec-2017 20:28    168K
<a href="Wget_Windows_Guide.pdf">Wget_Windows_Guide.pdf</a>                             25-Nov-2017 17:59    867K
<a href="rclone_guide.pdf">rclone_guide.pdf</a>                                   03-Sep-2018 23:37    315K
<a href="wget-noobs-guide.pdf">wget-noobs-guide.pdf</a>                               21-Dec-2017 20:29    168K
<a href="xbox-scene_Aug2014.7z">xbox-scene_Aug2014.7z</a>                              26-Oct-2017 23:09      1G
</pre><hr></body>
</html>`
