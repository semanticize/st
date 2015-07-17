import json
import subprocess


class Semanticizer:
    ''' Wrapper for semanticizest go implementation. '''

    def __init__(self, model='nl.go.model', stPath='./bin/semanticizest'):
        ''' Create an instance of Semanticizer.

        Arguments:
        model  -- Language model created by semanticizest-dumpparser
        stPath -- Path to semanticizest go implementation.
        '''
        args = [stPath, model]
        self.proc = subprocess.Popen(args, stdin=subprocess.PIPE,
                                     stdout=subprocess.PIPE)

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
        self.proc.stdin.write(json.dumps(sentence) + '\n\n')
        stdoutdata = self.proc.stdout.readline()
        return self._responseGenerator(stdoutdata)

    def _responseGenerator(self, data):
        # Should be a generator instead of returning an array ?
        dataJson = json.loads(data)
        return dataJson if dataJson is not None else []

    def __del__(self):
        # Will eventually get called
        self.proc.terminate()

    def __exit__(self):
        self.__del__()
