 
FROM ubuntu:18.04

ENV LANG=C.UTF-8 LC_ALL=C.UTF-8
ENV PATH /opt/conda/bin:$PATH

ENV NGINX_VERSION 1.17.8

RUN apt-get -y update

RUN apt-get install -y \
    build-essential git wget \
    cmake pkg-config ca-certificates openssl \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
    && apt-get clean \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN wget --quiet https://repo.anaconda.com/miniconda/Miniconda2-4.5.11-Linux-x86_64.sh -O ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    rm ~/miniconda.sh && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" >> ~/.bashrc && \
    echo "conda activate base" >> ~/.bashrc

# creating conda environment
COPY environment.yml / 
RUN conda env create -f environment.yml

RUN mkdir /proto
COPY proto/ /proto/
COPY rtsp_to_rtmp.py start.sh global_vars.py read_image.py archive.py disk_cleanup.py inmemory_buffer.py /

RUN chmod +x /start.sh

ENTRYPOINT ["./start.sh"]