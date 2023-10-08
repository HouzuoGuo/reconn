import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService } from './chat.module';
import { VoiceModelResponse, VoiceService } from './chat.module';

@Component({
  selector: 'tts',
  templateUrl: './tts.component.html',
})
export class TTSComponent implements OnInit {
  userID = 'howard';
  text = 'When the king saw Ester the queen standing in the courtyard, she won his favor; so the king extended the gold scepter in his hand toward Ester.';
  topK = '99';
  topP = '0.8';
  mineosP = '0.01';
  semanticTemp = '0.7';
  waveformTemp = '0.6';
  fineTemp = '0.5';

  statusMessage = '';
  speechBlob = new ReplaySubject<Blob>(1);
  speechBlobSrc: Observable<string>;

  constructor(private voiceService: VoiceService) {
    this.speechBlobSrc = this.speechBlob.pipe(
      map((input: Blob) => this.blobToUrl(input)),
    );
  }

  ngOnInit() {
    this.speechBlob.subscribe(v => console.log('@@@@@@ speech ', v));
  }

  blobToUrl(input: Blob) {
    const blob = new Blob([input], { type: "audio/wav" });
    const blobUrl = URL.createObjectURL(blob);
    return blobUrl;
  }

  ttsButtonClick() {
    this.statusMessage = 'Converting to speech, this may take a minute.';
    this.voiceService.textToSpeechRealTIme(this.userID, this.text, Number(this.topK), Number(this.topP), Number(this.mineosP), Number(this.semanticTemp), Number(this.waveformTemp), Number(this.fineTemp))
      .pipe(tap(_ => { this.statusMessage = 'Ready, give it a listen.'; }))
      .subscribe(this.speechBlob);
  }

}
