# sconfd.py
# Author: https://github.com/hanneslehmann
# Licence: free to use, no warranties!

import sys
import redis
import time
from datetime import datetime

FORMAT = '%Y%m%d%H%M%S'

def dumpdata(obj,sep, f):
    if type(obj) == dict:
        for k, v in obj.items():
            if hasattr(v, '__iter__'):
                print k
                dumpdata(v)
            else:
#                print '%s %s %s' % (k, sep, v)
                f.write(k + sep + v + '\n')
                f.flush()
    elif type(obj) == list:
        for v in obj:
            if hasattr(v, '__iter__'):
                dumpdata(v)
            else:
                print v
    else:
        print obj

myfiles = []
whatchlist = set([])

def main(argv):
  r = redis.StrictRedis(host='localhost', port=6379, db=1)
  c = redis.StrictRedis(host='localhost', port=6379, db=1)
  lookfor = 'config:'+str(argv[0])+':'
  # print "my keys: " + lookfor
  # first get a list of config/template files I am interested in
  print "Watching..."
  for val in c.keys(lookfor+"*"):
    k = val.split(':')
    # myfiles.append(k[2])
    whatchlist.add(k[0]+":"+k[1])
    whatchlist.add("template:"+k[2])
    #print k[0]+":"+k[1]+":"+k[2]
    #print "template:"+k[2]
  # output to confirm we are whatching
  print whatchlist
  # now subscribing to changes
  pubsub = r.pubsub()
  pubsub.psubscribe('__keyevent@1__:hset')
  # when change happened we need to check if change
  # was done on a file we are watching
  for msg in pubsub.listen():
    s = str(msg['data'])
    params = s.split(":")
    ck = ""
    # check if the key contains at least lvl 2
    if len(params)>=2:
      ck = params[0]+":"+ params[1]
    # check if the key is being whatched
    if ck in whatchlist:
      # print "I was watching you!"
      key_index = 0
      if params[0] == 'config':
        key_index = 2
      if params[0] == "template":
        key_index = 1     
      if key_index > 0:
        # construct the keys
        key_c = lookfor+params[key_index]+':content'
        key_m = lookfor+params[key_index]+':meta'
        key_t = "template:"+params[key_index]
        # get data from template
        sep = c.hget(key_t + ":meta",'seperator')
        com = c.hget(key_t + ":meta",'comment')
        data_template = c.hgetall(key_t + ":content")
        # get data from client config
        fp  = c.hget(key_m,'filepath')
        data_client = c.hgetall(key_c)
        # copy template to temporary location
        data = data_template.copy()
        data.update(data_client)
        print "writing change of "+ ck +" to file: " + fp
        f = open(fp, 'w')
        f.write(com + ' Updated by sconfd at: ' + datetime.now().strftime(FORMAT) + '\n')
        f.flush()
        dumpdata (data, sep, f)
        f.close

if __name__ == "__main__":
    main(sys.argv[1:])
