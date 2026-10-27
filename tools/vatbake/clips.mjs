// Procedural animation clips for the grab3d 19-bone rig (hips, spine, chest,
// neck, head, shoulderL/R, upperArmL/R, lowerArmL/R, handL/R, upperLegL/R,
// lowerLegL/R, footL/R). Characters face +Z, left side is +X.
//
// Each key is an absolute pose expressed as WORLD-space rotations applied on
// top of the rest pose (idle, frame 0), so we don't depend on each bone's
// local axes. World axes: X = character's left, Y = up, Z = forward.
//   rotate about X by negative degrees: limb swings forward/up (arms), body leans back
//   rotate about X by positive degrees: body leans forward
//   rotate about Y by positive degrees: turns right side forward

import * as THREE from "three";

const X = [1, 0, 0];
const Y = [0, 1, 0];
const Z = [0, 0, 1];

/** Sword: wind up overhead, strike down-forward, recover. */
export function swing() {
  return {
    duration: 0.6,
    keys: [
      { t: 0.0, ops: [] },
      {
        t: 0.22, // wind-up: sword raised behind the head, right shoulder back
        ops: [
          ["chest", Y, -22],
          ["upperArmR", X, -155],
          ["lowerArmR", X, -30],
          ["upperArmL", X, -45], // shield up
        ],
      },
      {
        t: 0.36, // strike
        ops: [
          ["spine", X, 12],
          ["chest", Y, 25],
          ["upperArmR", X, -55],
          ["lowerArmR", X, -10],
          ["upperArmL", X, -50],
        ],
      },
      {
        t: 0.45,
        ops: [
          ["spine", X, 8],
          ["chest", Y, 18],
          ["upperArmR", X, -45],
          ["upperArmL", X, -40],
        ],
      },
      { t: 0.6, ops: [] },
    ],
  };
}

/** Bow: raise the bow arm forward, draw the string to the face, release. */
export function bow() {
  const draw = [
    ["chest", Y, -35], // sideways stance, bow shoulder forward
    ["upperArmL", X, -88], // bow arm straight forward
    ["upperArmR", X, -85], // drawing arm forward...
    ["lowerArmR", Y, 115], // ...elbow bent, hand pulled back to the face
  ];
  return {
    duration: 0.8,
    keys: [
      { t: 0.0, ops: [] },
      { t: 0.25, ops: draw },
      { t: 0.45, ops: draw }, // hold
      {
        t: 0.52, // release: drawing hand snaps back and out
        ops: [
          ["chest", Y, -35],
          ["upperArmL", X, -88],
          ["upperArmR", X, -70],
          ["upperArmR", Y, -20],
          ["lowerArmR", Y, 135],
        ],
      },
      { t: 0.8, ops: [] },
    ],
  };
}

/** Death: knees buckle, falls on the back, small bounce. Holds the last frame. */
export function die() {
  return {
    duration: 1.0,
    keys: [
      { t: 0.0, ops: [] },
      {
        t: 0.15, // hit: head snaps back, knees bend
        ops: [
          ["spine", X, -10],
          ["head", X, -15],
          ["upperLegL", X, -20],
          ["upperLegR", X, -20],
          ["lowerLegL", X, 35],
          ["lowerLegR", X, 35],
        ],
        hipsY: 0.85,
      },
      {
        t: 0.55, // falling backwards
        ops: [
          ["hips", X, -60],
          ["spine", X, -10],
          ["upperArmL", Z, 40],
          ["upperArmR", Z, -40],
          ["lowerLegL", X, 30],
          ["lowerLegR", X, 30],
        ],
        hipsY: 0.45,
      },
      {
        t: 0.72, // hits the ground
        ops: [
          ["hips", X, -88],
          ["head", X, 10],
          ["upperArmL", Z, 70],
          ["upperArmR", Z, -70],
          ["lowerLegL", X, 15],
          ["lowerLegR", X, 15],
        ],
        hipsY: 0.16,
      },
      {
        t: 0.82, // bounce
        ops: [
          ["hips", X, -82],
          ["head", X, 5],
          ["upperArmL", Z, 75],
          ["upperArmR", Z, -75],
          ["lowerLegL", X, 20],
          ["lowerLegR", X, 20],
        ],
        hipsY: 0.2,
      },
      {
        t: 1.0, // lying still
        ops: [
          ["hips", X, -90],
          ["upperArmL", Z, 80],
          ["upperArmR", Z, -80],
          ["lowerLegL", X, 12],
          ["lowerLegR", X, 12],
        ],
        hipsY: 0.15,
      },
    ],
  };
}

export const generators = { swing, bow, die };

/**
 * Builds a THREE.AnimationClip from a procedural definition.
 * rest: Map<boneName, {position, quaternion}> (local), bones: Map<boneName, Bone>.
 */
export function buildClip(name, def, rest, bones, root) {
  const touched = new Set(["hips"]);
  for (const k of def.keys) for (const [b] of k.ops) touched.add(b);

  const times = def.keys.map((k) => k.t);
  const quatValues = new Map([...touched].map((b) => [b, []]));
  const hipsPos = [];
  const tmpQ = new THREE.Quaternion();
  const worldQ = new THREE.Quaternion();
  const parentQ = new THREE.Quaternion();

  for (const key of def.keys) {
    resetToRest(rest, bones);
    const restHips = rest.get("hips").position;
    bones.get("hips").position.set(restHips.x, key.hipsY ?? restHips.y, restHips.z);
    root.updateMatrixWorld(true);

    for (const [boneName, axis, deg] of key.ops) {
      const bone = bones.get(boneName);
      if (!bone) throw new Error(`clip ${name}: unknown bone ${boneName}`);
      bone.getWorldQuaternion(worldQ);
      tmpQ.setFromAxisAngle(new THREE.Vector3(...axis), THREE.MathUtils.degToRad(deg));
      worldQ.premultiply(tmpQ); // new world rotation
      bone.parent.getWorldQuaternion(parentQ);
      bone.quaternion.copy(parentQ.invert().multiply(worldQ));
      root.updateMatrixWorld(true);
    }

    for (const b of touched) quatValues.get(b).push(...bones.get(b).quaternion.toArray());
    hipsPos.push(...bones.get("hips").position.toArray());
  }
  resetToRest(rest, bones);

  const tracks = [new THREE.VectorKeyframeTrack("hips.position", times, hipsPos)];
  for (const [b, values] of quatValues) {
    tracks.push(new THREE.QuaternionKeyframeTrack(`${b}.quaternion`, times, values));
  }
  return new THREE.AnimationClip(name, def.duration, tracks);
}

function resetToRest(rest, bones) {
  for (const [name, r] of rest) {
    const b = bones.get(name);
    b.position.copy(r.position);
    b.quaternion.copy(r.quaternion);
  }
}
