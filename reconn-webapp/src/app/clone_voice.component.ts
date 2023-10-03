import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, startWith } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { VoiceModelResponse, VoiceService } from './voice.service';

@Component({
  selector: 'clone-voice',
  templateUrl: './clone_voice.component.html',
})
export class CloneVoiceComponent implements OnInit {
  @ViewChild('cloneButton', { static: true }) cloneButton!: ElementRef;

  userID = '';
  recordingInProgress = false;

  ngOnInit() {
    fromEvent(this.cloneButton.nativeElement, 'click').subscribe(a => console.log(this.userID));
  }
}
