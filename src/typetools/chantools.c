// Low-level channel tools

#include "runtime.h"

// Structure defs mirrored from go/src/pkg/runtime/chan.c
typedef struct  WaitQ WaitQ;
typedef struct  SudoG SudoG;
struct  SudoG
{
  G*  g;    // g and selgen constitute
  uint32  selgen;   // a weak pointer to g
  SudoG*  link;
  byte* elem;   // data element
};

struct  WaitQ
{
  SudoG*  first;
  SudoG*  last;
};

struct  Hchan
{
  uint32  qcount;     // total data in the q
  uint32  dataqsiz;   // size of the circular q
  uint16  elemsize;
  bool  closed;
  uint8 elemalign;
  Alg*  elemalg;    // interface for element type
  uint32  sendx;      // send index
  uint32  recvx;      // receive index
  WaitQ recvq;      // list of recv waiters
  WaitQ sendq;      // list of send waiters
  Lock;
};
#define chanbuf(c, i) ((byte*)((c)+1)+(uintptr)(c)->elemsize*(i))

void ·ChanDebug(uint32 t, Hchan* c) {
  runtime·lock(c);
  runtime·printf("Type: %x, ChanPtr: %p\n", t, c);
  runtime·printf("QSize:%d, Elem:%d\n", c->dataqsiz, c->elemsize);
  runtime·printf("Value count: %d\n", c->qcount);
  if (c->dataqsiz < 1) {
    runtime·printf("Cannot peek on an unbuffered channel\n");
    return;
  }
  runtime·printf("Peeking at [recv:%d send:%d %d/%d]\n", c->recvx, c->sendx, c->qcount, c->dataqsiz);
  runtime·unlock(c);
}

// Main batching function
// Read up to minnum values from the channel into a new array
void ·ChanRead(uint32 t, Hchan* c, uint32 minnum, byte* ret, uint32 len, uint32 size) {
  runtime·lock(c);
  runtime·printf("%d <= %d?\n", minnum, c->qcount);
  len = c->qcount;
  FLUSH(&len);
  if (c->qcount < minnum) {
    ret = nil;
    FLUSH(&ret);
    runtime·unlock(c);
    return;
  }
  size = c->elemsize * c->qcount;
  FLUSH(&size);
  ret = (byte*) runtime·mal(size);
  FLUSH(&ret);
  runtime·printf("ChanPtr: %p, %x\n", c, t);
  runtime·printf("QSize:%d, Elem:%d\n", c->dataqsiz, c->elemsize);
  runtime·printf("Value count: %d\n", c->qcount);
  if (c->dataqsiz < 1) {
    runtime·printf("Cannot peek on an unbuffered channel\n");
    runtime·unlock(c);
    return;
  }
  // Copy values from the channel
  runtime·printf("Reading [recv:%d send:%d %d/%d]\n", c->recvx, c->sendx, c->qcount, c->dataqsiz);
  // TODO (performance): change to a memcpy
  int32 j = 0;
  for (int32 i = c->recvx; i < c->sendx && i < c->dataqsiz; i++) {
    c->elemalg->copy(c->elemsize, (ret + (j * c->elemsize)), chanbuf(c, i));
    j++;
  }
  if (c->sendx < c->recvx) {
    // Wrapped around
    for (int32 i = 0; i < c->sendx; i++) {
      c->elemalg->copy(c->elemsize, (ret + (j * c->elemsize)), chanbuf(c, i));
      j++;
    }
  }
  c->recvx = c->sendx = 0;
  c->qcount = 0;
  runtime·unlock(c);
}
