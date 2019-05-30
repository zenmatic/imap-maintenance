FROM ubuntu:18.10

RUN apt-get update -y \
	&& apt-get install -y dovecot-imapd

RUN groupadd vmail \
	&& useradd -g vmail vmail \
	&& mkdir -p /home/vmail \
	&& chown vmail:vmail /home/vmail

RUN mkdir -p /var/log/rawlog
ADD dovecot.conf /etc/dovecot/dovecot.conf
ADD run.sh /run.sh

CMD bash /run.sh
