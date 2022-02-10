#FROM $com.sap.sles.base
FROM python:3.6.4-slim-stretch

RUN apt-get update && \
    apt install -y python3-pip
RUN pip3 install mysql-connector-python==8.0.12
RUN pip3 install pandas
RUN pip3 install tornado
RUN pip3 install pympler


RUN groupadd -g 1972 vflow && useradd -g 1972 -u 1972 -m vflow
USER 1972:1972
WORKDIR /home/vflow
ENV HOME=/home/vflow
#RUN pip3.6 install --user mysql-connector-python=8.0.12