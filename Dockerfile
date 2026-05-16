# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache \
    cmake \
    ninja \
    make \
    gcc \
    g++ \
    musl-dev \
    linux-headers \
    git \
    openssl-dev \
    pkgconfig

# Build liboqs from source
RUN git clone --depth 1 --branch main https://github.com/open-quantum-safe/liboqs.git /liboqs && \
    cmake -S /liboqs -B /liboqs/build \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=ON \
        -DOQS_BUILD_ONLY_LIB=ON \
        -DOQS_USE_OPENSSL=OFF \
        -GNinja && \
    cmake --build /liboqs/build && \
    cmake --install /liboqs/build

# Create missing liboqs-go.pc
RUN printf 'prefix=/usr/local\nexec_prefix=${prefix}\nlibdir=${exec_prefix}/lib\nincludedir=${prefix}/include\n\nName: liboqs-go\nDescription: Open Quantum Safe liboqs Go wrapper\nVersion: 0.15.0\nLibs: -L${libdir} -loqs\nCflags: -I${includedir}\n' > /usr/local/lib/pkgconfig/liboqs-go.pc && cat /usr/local/lib/pkgconfig/liboqs-go.pc

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN PKG_CONFIG_PATH=/usr/local/lib/pkgconfig CGO_ENABLED=1 GOOS=linux go build -o securevault .

# Run stage
FROM alpine:3.19

RUN apk add --no-cache libstdc++ openssl

RUN addgroup -S vault && adduser -S -G vault vault

WORKDIR /app

COPY --from=builder /build/securevault .
COPY --from=builder /usr/local/lib/liboqs.so* /usr/local/lib/
COPY templates/ ./templates/

RUN mkdir -p uploads && chown -R vault:vault /app
RUN ldconfig /usr/local/lib 2>/dev/null || true

USER vault

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:5000 || exit 1

CMD ["./securevault"]
