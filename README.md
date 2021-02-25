# wpaste
## Description
wpaste is a service for easy share your code with others. Just send code and share link.

[Source](https://github.com/waika28/wpaste)

**WARNING: DO NOT USE IMPORTANT PASSWORDS AND DO NOT UPLOAD IMPORTANT FILES ONLY WITH SERVER PASSWORD. IT IS NOT SECURE.**

## Using

1. `cat file.txt | curl -F 'f=<-' %addr_to_server%`
2. Share

| Method   | Path   | Param           | Result                                            |
|:--------:|:------:|-----------------|---------------------------------------------------|
|GET       |/       |                 |This README file                                   |
|GET       |/\<name>|                 |File by name                                       |
|GET       |/\<name>|ap=pass          |Protected file by name                             |
|POST      |/       |f=file           |Random name for access to your file*               |
|POST      |/       |f=f, e=3600      |After 3600sec (1 hour) file will not be available**|
|POST      |/       |f=f, name=Myname |File with access by specifed name                  |
|POST      |/       |f=f, ap=pass     |Access to file by password                         |
|POST      |/       |f=f, ep=pass     |Access to edit file                                |
|PUT       |/\<name>|f=f, ep=pass     |Change content to f                                |
|DELETE    |/\<name>|f=f, ep=pass     |Remove file                                        |

*by default files haven't expires  
**expired file will be permanently deleted after 4 hours, until that time, it will respond with code 410

For really data protection use [GnuPG](https://gnupg.org/)/[ccrypt](http://ccrypt.sourceforge.net/)
### Example:
```bash
echo "secret text" | ccrypt -e -K passwd | base64 | curl -F "f=<-" %addr_to_server%

curl -s %addr_to_server%/dBd | base64 -d | ccrypt -d -K passwd
```

## LICENSE
wpaste - easy code sharing  
Copyright (C) 2020  Evgeniy Rybin

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.