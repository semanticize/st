import json
import subprocess
import tempfile
import urllib2


class SemanticizerClient:
    ''' HTTP client for semanticizest go implementation. '''
    def __init__(self, serverURL):
        ''' Create an instance of SemanticizerClient.

        Arguments:
        serverURL  -- URL of server this client connects to.
        '''
        service = 'all'
        self._url = serverURL + '/' + service

    def all_candidates(self, sentence):
        ''' Given a sentence, generate a list of candidate entity links.

        Returns a list of candidate entity links, where each candidate entity
        is represented by a dictionary containing:
         - target     -- Title of the target link
         - offset     -- Offset of the anchor on the original sentence
         - length     -- Length of the anchor on the original sentence
         - commonness -- commonness of the link
         - senseprob  -- probability of the link
         - linkcount
         - ngramcount
        '''
        req = urllib2.Request(self._url)
        resp = urllib2.urlopen(req, json.dumps(sentence))
        respLines = resp.readlines()
        candidates = json.loads(''.join(respLines))
        return candidates


class SemanticizerServer:
    ''' HTTP server wrapper for semanticizest go implementation. '''
    def __init__(self, model='nl.go.model', stPath='./bin/semanticizest'):
        ''' Create an instance of SemanticizerServer.

        Arguments:
        model  -- Language model created by semanticizest-dumpparser
        stPath -- Path to semanticizest go implementation.
        '''
        portfile = tempfile.NamedTemporaryFile()
        args = [stPath, '--http=:0', '--portfile='+portfile.name, model]
        self._proc = subprocess.Popen(args, stderr=subprocess.PIPE)

        # Wait for port file...
        port = ''
        while len(port) == 0:
            port = portfile.file.readline().strip()
        self._gourl = 'http://localhost:' + port

    def getURL(self):
        ''' Retrieve the URL of this server. '''
        return self._gourl

    def stop(self):
        ''' Stop the server. Terminate subprocess. '''
        self._proc.terminate()
