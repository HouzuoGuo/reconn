import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, startWith } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { VoiceModelResponse, VoiceService } from './voice.service';
import { AudioRecorderService } from './audio_recorder_module';

@Component({
  selector: 'clone-voice',
  templateUrl: './clone_voice.component.html',
})
export class CloneVoiceComponent implements OnInit {
  @ViewChild('cloneButton', { static: true }) cloneButton!: ElementRef;

  userID = '';
  recordingInProgress = false;

  constructor(readonly recorderService: AudioRecorderService) {

  }

  ngOnInit() {
    fromEvent(this.cloneButton.nativeElement, 'click').subscribe(a => console.log(this.userID));
  }
}
