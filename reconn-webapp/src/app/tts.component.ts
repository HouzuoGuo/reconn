import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, startWith } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { VoiceModelResponse, VoiceService } from './voice.service';

@Component({
  selector: 'tts',
  templateUrl: './tts.component.html',
})
export class TTSComponent implements OnInit {
  ngOnInit() {
  }
}
