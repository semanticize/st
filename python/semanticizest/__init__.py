import json
import os
import os.path
from shutil import rmtree
import subprocess
from tempfile import mkdtemp
import urllib2


__all__ = ['Client', 'Semanticizer']


class Client(object):
    """Client for running semanticizest REST server.

    Parameters
    ----------
    url : string
        Base URL of running semanticizest REST endpoint.
        See SemanticizerServer for a way to start a server from Python.

    Attributes
    ----------
    url : string
        URL of REST endpoint, as passed to the initializer.
    """

    def __init__(self, url):
        self.url = url

    def _call(self, method, text):
        """Call REST method."""
        url = os.path.join(self.url, method)
        req = urllib2.Request(url)
        response = urllib2.urlopen(req, json.dumps(text)).read()
        return json.loads(response)

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
        return self._call('all', sentence)


class Semanticizer(Client):
    """Entity linker.

    This is actually a Client that starts its own server as a subprocess.

    Parameters
    ----------
    model : string
        Path to model file, as produced by semanticizest-dumpparser.
    serverpath : string, optional
        Path to (or name of) semanticizest REST server binary.
        If omitted, tries to find the binary in $GOPATH/bin.
    """

    def __init__(self, model, serverpath=None):
        if serverpath == None:
            serverpath = os.path.join(os.environ['GOPATH'],
                                      'bin', 'semanticizest')
        d = mkdtemp(prefix='semanticizest-py')
        try:
            portfifo = os.path.join(d, 'portfifo')
            os.mkfifo(portfifo)

            args = [serverpath, '--http=:0', '--portfile=' + portfifo, model]
            # TODO start a thread that consumes stderr and acts on it.
            proc = subprocess.Popen(args)

            try:
                with open(portfifo) as f:
                    port = int(f.read().strip())
            except:
                proc.terminate()
                raise

            self._proc = proc
            self.url = 'http://localhost:%d' % port

        finally:
            rmtree(d, ignore_errors=True)

    def __del__(self):
        self._proc.terminate()  # XXX Is this necessary?
